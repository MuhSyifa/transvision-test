package controllers

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"test-transvision/helper"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

func Register(c *gin.Context, dbs helper.DBStruct) {
	jsonData, _ := ioutil.ReadAll(c.Request.Body)
	postdata := map[string]interface{}{}
	json.Unmarshal(jsonData, &postdata)

	username := postdata["username"]
	fullname := postdata["fullname"]
	email := postdata["email"]
	pasword := postdata["password"]

	sqlCheckEmail := `SELECT * FROM m_users WHERE other_data->>'email'='` + cast.ToString(email) + `'`
	rowEmailAlready := dbs.DatabaseQuerySingleRow(sqlCheckEmail)

	if cast.ToString(username) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Username cannot be empty",
		})
		return
	} else if cast.ToString(fullname) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Fullname cannot be empty",
		})
		return
	} else if cast.ToString(email) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Email cannot be empty",
		})
		return
	} else if !helper.ValidFormatEmail(cast.ToString(email)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Format Email not valid",
		})
		return
	} else if len(rowEmailAlready) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Email already used",
		})
		return
	} else if cast.ToString(pasword) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Password cannot be empty",
		})
		return
	}

	// fmt.Println("password : ", helper.Pass2Hash(cast.ToString(pasword)))
	userID := "USR" + uuid.New().String()

	now := time.Now()
	created_at := now.Format("2006-01-02 15:04:05")
	other := map[string]interface{}{
		"email":      email,
		"fullname":   fullname,
		"is_active":  0,
		"created_at": created_at,
		"created_by": "",
		"is_deleted": false,
	}
	otherData, _ := json.Marshal(other)

	sqlInsert := `
					INSERT INTO m_users 
					("user_id","username","password","level","other_data") 
					VALUES($1,$2,$3,$4,$5)
				`

	_, errDE := dbs.Dbx.Exec(sqlInsert,
		userID,
		username,
		helper.Pass2Hash(cast.ToString(pasword)),
		2,
		cast.ToString(otherData),
	)

	uniqID := userID+" "+cast.ToString(email)
	encriptUID := b64.StdEncoding.EncodeToString([]byte(uniqID))

	if errDE != nil {
		fmt.Println(errDE)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Gagal Register",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"link_verifikasi": viper.GetString("base_url") + "auth/verifikasi/"+encriptUID,
		},
		"message": "Sukses Register",
	})
}

func Verifikasi(c *gin.Context, dbs helper.DBStruct) {
	uid := c.Param("uid")
	uidDec, _ := b64.StdEncoding.DecodeString(uid)

	newUID := strings.Split(string(uidDec)," ")
	
	now := time.Now()
	updated_at := now.Format("2006-01-02 15:04:05")
	other := map[string]interface{}{
		"is_active":     1,
		"updated_at":    updated_at,
		"updated_by":    newUID[1],
		"is_deleted":    false,
	}
	otherData, _ := json.Marshal(other)

	query := `UPDATE m_users SET other_data=$1 WHERE user_id='` + newUID[0] + `'`
	_, err := dbs.Dbx.Exec(query, cast.ToString(otherData))
	if err != nil {
		fmt.Println("error on update : ", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Gagal Verifikasi",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sukses Verifikasi",
	})
}

func login(c *gin.Context, dbs helper.DBStruct) {
	jsonData, _ := ioutil.ReadAll(c.Request.Body)
	postdata := map[string]interface{}{}
	json.Unmarshal(jsonData, &postdata)

	username := postdata["username"]
	pasword := helper.Pass2Hash(cast.ToString(postdata["password"]))

	sqlCheckUser := `SELECT * FROM m_users WHERE username='`+cast.ToString(username)+`'`
	rowCheckUser := dbs.DatabaseQuerySingleRow(sqlCheckUser)

	if pasword != cast.ToString(rowCheckUser["password"]) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Username or Password wrong",
		})
		return
	}


}
