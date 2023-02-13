package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"test-transvision/helper"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"
)

func ProductList(c *gin.Context, dbs helper.DBStruct) {
	sql := `SELECT
				product_id,
				name,
				other_data->>'desc' as desc,
				other_data->>'price' as price,
				other_data->>'stock' as stock,
				other_data->>'product_image' as product_image
			FROM m_product WHERE other_data->>'is_deleted'='false'
			`

	rows := dbs.DatabaseQueryRows(sql)

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   rows,
	})
}

func AllProduct(c *gin.Context, dbs helper.DBStruct) {
	sql := `SELECT
				product_id,
				name,
				other_data->>'desc' as desc,
				other_data->>'price' as price,
				other_data->>'stock' as stock,
				other_data->>'product_image' as product_image
			FROM m_product WHERE other_data->>'is_deleted'='false'
			`

	rows := dbs.DatabaseQueryRows(sql)

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   rows,
	})
}

func ProductCreate(c *gin.Context, dbs helper.DBStruct) {
	jsonData, _ := ioutil.ReadAll(c.Request.Body)
	postdata := map[string]interface{}{}
	json.Unmarshal(jsonData, &postdata)

	data := map[string]interface{}{
		"name":          postdata["name"],
		"desc":          postdata["description"],
		"stock":         postdata["stock"],
		"price":         postdata["price"],
		"product_image": postdata["product_image"],
	}

	productID := uuid.New().String()
	now := time.Now()
	created_at := now.Format("2006-01-02 15:04:05")
	other := map[string]interface{}{
		"desc":          data["desc"],
		"stock":         data["stock"],
		"price":         data["price"],
		"product_image": data["product_image"],
		"created_at":    created_at,
		"created_by":    "",
		"is_deleted":    false,
	}
	otherData, _ := json.Marshal(other)

	sqlInsert := `
					INSERT INTO m_product 
					("product_id","name","other_data") 
					VALUES($1,$2,$3)
				`

	_, errDE := dbs.Dbx.Exec(sqlInsert,
		productID,
		data["name"],
		cast.ToString(otherData),
	)

	if errDE != nil {
		fmt.Println(errDE)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Gagal Simpan Data Product",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sukses Simpan Data Product",
	})
}

func ProductUpdate(c *gin.Context, dbs helper.DBStruct) {
	productID := c.Param("uuid")
	jsonData, _ := ioutil.ReadAll(c.Request.Body)
	postdata := map[string]interface{}{}
	json.Unmarshal(jsonData, &postdata)

	data := map[string]interface{}{
		"name":          postdata["name"],
		"desc":          postdata["description"],
		"stock":         postdata["stock"],
		"price":         postdata["price"],
		"product_image": postdata["product_image"],
	}

	now := time.Now()
	updated_at := now.Format("2006-01-02 15:04:05")
	other := map[string]interface{}{
		"desc":          data["desc"],
		"stock":         data["stock"],
		"price":         data["price"],
		"product_image": data["product_image"],
		"updated_at":    updated_at,
		"updated_by":    "",
		"is_deleted":    false,
	}
	otherData, _ := json.Marshal(other)

	query := `UPDATE m_product SET name=$1, other_data=other_data || $2 WHERE product_id='` + productID + `'`
	_, err := dbs.Dbx.Exec(query, data["name"], cast.ToString(otherData))
	if err != nil {
		fmt.Println("error on update : ", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Gagal Update Data Product",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sukses Update Data Product",
	})
}

func ProductGetByID(c *gin.Context, dbs helper.DBStruct) {
	productID := c.Param("uuid")
	sql := `SELECT
				product_id,
				name,
				other_data->>'desc' as desc,
				other_data->>'price' as price,
				other_data->>'stock' as stock,
				other_data->>'product_image' as product_image
			FROM m_product WHERE other_data->>'is_deleted'='false' AND product_id='`+productID+`'
			`

	row := dbs.DatabaseQuerySingleRow(sql)

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   row,
	})
}

func ProductDelete(c *gin.Context, dbs helper.DBStruct) {
	productID := c.Param("uuid")

	now := time.Now()
	deleted_at := now.Format("2006-01-02 15:04:05")
	other := map[string]interface{}{
		"deleted_at":    deleted_at,
		"deleted_by":    "",
		"is_deleted":    true,
	}
	otherData, _ := json.Marshal(other)

	query := `UPDATE m_product SET other_data=other_data || $1 WHERE product_id='` + productID + `'`
	_, err := dbs.Dbx.Exec(query, cast.ToString(otherData))
	if err != nil {
		fmt.Println("error on update : ", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Gagal Delete Data Product",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sukses Delete Data Product",
	})
}
