package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"test-transvision/helper"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

func ProductOrder(c *gin.Context, dbs helper.DBStruct) {
	jsonData, _ := ioutil.ReadAll(c.Request.Body)
	postdata := map[string]interface{}{}
	json.Unmarshal(jsonData, &postdata)

	now := time.Now()
	orderDate := now.Format("2006-01-02 15:04:05")

	// fmt.Println(postdata)
	checkoutID := cast.ToSlice(postdata["checkout_id"])
	arrCheckoutID := []string{}
	for _, v := range checkoutID {
		arrCheckoutID = append(arrCheckoutID, cast.ToString(v))
	}

	commaSepcheckoutID := "'" + strings.Join(arrCheckoutID, "', '") + "'"
	sql := `SELECT
				a.checkout_id,
				a.other_data->>'qty' as qty,
				a.user_id,
				a.product_id,
				b.other_data->>'price' as price
			FROM t_checkout a
			JOIN m_product b ON a.product_id=b.product_id 
			WHERE checkout_id IN (` + commaSepcheckoutID + `)`
	rows := dbs.DatabaseQueryRows(sql)

	// fmt.Println(rows)
	var resultAmount float64
	for _, v := range rows {
		resultAmount += cast.ToFloat64(v["price"]) * cast.ToFloat64(v["qty"])
	}


	////////// t_order_header //////////
	orderID := "ORD" + uuid.New().String()

	other := map[string]interface{}{
		"amount":          resultAmount,
		"ship_name":       postdata["ship_name"],
		"ship_address":    postdata["ship_address"],
		"ship_city":       postdata["ship_city"],
		"ship_state":      postdata["ship_state"],
		"ship_country":    postdata["ship_country"],
		"ship_phone":      postdata["ship_phone"],
		"tracking_number": postdata["tracking_number"],
		"payment_method":  postdata["payment_method"],
	}
	otherData, _ := json.Marshal(other)

	sqlInsert := `
					INSERT INTO t_order_header 
					("order_id","user_id","order_date","other_data") 
					VALUES($1,$2,$3,$4)
				`

	_, errDE := dbs.Dbx.Exec(sqlInsert,
		orderID,
		rows[0]["user_id"],
		orderDate,
		cast.ToString(otherData),
	)

	if errDE != nil {
		fmt.Println(errDE)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Gagal Simpan Data Order Header",
		})
		return
	}

	////////// t_order_detail //////////
	queryInsert := `INSERT INTO t_order_detail(order_detail_id,order_id,product_id,other_data) VALUES `
	insertparams := []interface{}{}

	for i := 0; i < len(rows); i++ {
		orderDetailID := "ORD-DTL" + uuid.New().String() + cast.ToString(i)
		cOD, _ := json.Marshal(map[string]interface{}{
			"qty": rows[i]["qty"],
		})

		p1 := i * 4
		queryInsert += fmt.Sprintf("($%d,$%d,$%d,$%d),", p1+1, p1+2, p1+3, p1+4)
		insertparams = append(insertparams, cast.ToString(orderDetailID), orderID, rows[i]["product_id"], cOD)
	}

	queryInsert = queryInsert[:len(queryInsert)-1]
	_, errInsert := dbs.Dbx.Exec(queryInsert, insertparams...)

	statusInsert := true
	if errInsert != nil {
		statusInsert = false
		fmt.Println("err : ", errInsert.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Error Order Product",
		})
		return
	}

	if statusInsert {
		query_del := `DELETE FROM t_checkout WHERE checkout_id IN (` + commaSepcheckoutID + `)`
		_, err_del := dbs.Dbx.Exec(query_del)
		if err_del != nil {
			fmt.Println(err_del.Error())
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "Success Order Product",
	})
}

func ProductCheckout(c *gin.Context, dbs helper.DBStruct) {
	jsonData, _ := ioutil.ReadAll(c.Request.Body)
	postdata := map[string]interface{}{}
	json.Unmarshal(jsonData, &postdata)

	// fmt.Println(postdata)

	arrProduct := cast.ToSlice(postdata["product"])
	arrProductID := []string{}
	outputQty := map[string]map[string]interface{}{}
	for _, v := range arrProduct {
		product := cast.ToStringMap(v)
		productID := product["product_id"]

		arrProductID = append(arrProductID, cast.ToString(productID))
		outputQty[cast.ToString(productID)] = product
	}

	commaSepProductID := "'" + strings.Join(arrProductID, "', '") + "'"

	sql := `SELECT * FROM m_product WHERE product_id IN (` + commaSepProductID + `) AND other_data->>'is_deleted'='false'`
	rows := dbs.DatabaseQueryRows(sql)

	// fmt.Println("rows : ", rows)
	datas := []map[string]interface{}{}
	for _, v := range rows {
		qty := outputQty[cast.ToString(v["product_id"])]["qty"]
		res := map[string]interface{}{
			"product_id": v["product_id"],
			"name":       v["name"],
			"qty":        qty,
		}
		datas = append(datas, res)
	}

	jwtToken := strings.Split(c.GetHeader("Authorization"), " ")

	JWTTokenSecret := viper.GetString("jwt.token_secret")
	token, err := jwt.Parse(jwtToken[1], func(token *jwt.Token) (interface{}, error) {
		if _, OK := token.Method.(*jwt.SigningMethodHMAC); !OK {
			return nil, errors.New("bad signed method received")
		}
		return []byte(JWTTokenSecret), nil
	})

	if err != nil {
		fmt.Println("error", err.Error())
		// return nil, errors.New("bad jwt token")
	}

	claims, _ := token.Claims.(jwt.MapClaims)

	userID := claims["user_id"]
	now := time.Now()
	checkoutDate := now.Format("2006-01-02 15:04:05")

	// fmt.Println("jwtToken : ", jwtToken)
	// fmt.Println("token : ", token)
	// fmt.Println("userID : ", userID)

	queryInsert := `INSERT INTO t_checkout(user_id,product_id,checkout_date,other_data,checkout_id) VALUES `
	insertparams := []interface{}{}

	for i := 0; i < len(datas); i++ {
		checkoutID := "CHC" + uuid.New().String() + cast.ToString(i)
		cOD, _ := json.Marshal(map[string]interface{}{
			"qty": datas[i]["qty"],
		})

		p1 := i * 5
		queryInsert += fmt.Sprintf("($%d,$%d,$%d,$%d,$%d),", p1+1, p1+2, p1+3, p1+4, p1+5)
		insertparams = append(insertparams, cast.ToString(userID), datas[i]["product_id"], checkoutDate, cOD, checkoutID)
	}

	queryInsert = queryInsert[:len(queryInsert)-1]
	_, errInsert := dbs.Dbx.Exec(queryInsert, insertparams...)

	if errInsert != nil {
		fmt.Println("err : ", errInsert.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Error Checkout Product",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"data":    datas,
		"message": "Success Checkout Product",
	})
}
