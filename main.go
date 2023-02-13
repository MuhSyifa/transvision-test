package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"test-transvision/controllers"
	"test-transvision/helper"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

func main() {
	//---- READ CONFIG JSON ----
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.SetConfigName("app.conf")

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
	}

	username := viper.GetString("database.user")
	password := viper.GetString("database.password")
	database := viper.GetString("database.name")
	host := viper.GetString("database.host")
	port := viper.GetInt("database.port")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, username, password, database)
	db, err := sqlx.Connect("postgres", psqlInfo)
	if err != nil {
		fmt.Println("Error Connecting DB => ", err)
		os.Exit(0)
	}
	defer db.Close()

	maxLifetime, _ := time.ParseDuration(viper.GetString("database.max_lifetime_connection") + "s")
	db.SetMaxIdleConns(viper.GetInt("database.max_idle_connection"))
	db.SetConnMaxLifetime(maxLifetime)
	dbs := helper.DBStruct{Dbx: db}

	fmt.Println(dbs)

	//---- ROUTING -----
	router := gin.New()
	router.Use(cors.Default())

	Routing(router, dbs)

	pprof.Register(router)

	//---- RUNNING SERVER WITH PORT -----
	s := &http.Server{
		Addr:    ":" + viper.GetString("server.port"),
		Handler: router,
	}
	fmt.Println("Server running on port:", viper.GetString("server.port"))
	s.ListenAndServe()
}

func Routing(router *gin.Engine, dbs helper.DBStruct) {
	router.GET("/", func(c *gin.Context) { controllers.Home(c) })

	routerProduct := router.Group("product")
	routerProduct.Use(jwtTokenCheck)
	routerProduct.Use(privateACLCheck) // validate akses role administrator
	{
		routerProduct.GET("/list", func(c *gin.Context) { controllers.ProductList(c, dbs) })
		routerProduct.GET("/get/:uuid", func(c *gin.Context) { controllers.ProductGetByID(c, dbs) })
		routerProduct.POST("/create", func(c *gin.Context) { controllers.ProductCreate(c, dbs) })
		routerProduct.POST("/update/:uuid", func(c *gin.Context) { controllers.ProductUpdate(c, dbs) })
		routerProduct.DELETE("/delete/:uuid", func(c *gin.Context) { controllers.ProductDelete(c, dbs) })
	}

	routerPengguna := router.Group("")
	routerPengguna.Use(jwtTokenCheck)
	routerPengguna.Use(privatePenggunaCheck) // validate akses role pengguna
	{
		routerPengguna.GET("/list-product", func(c *gin.Context) { controllers.AllProduct(c, dbs) })
		routerPengguna.POST("/checkout-product", func(c *gin.Context) { controllers.ProductCheckout(c, dbs) })
		routerPengguna.POST("/order-product", func(c *gin.Context) { controllers.ProductOrder(c, dbs) })
	}

	routerAuth := router.Group("auth")
	{
		routerAuth.POST("/register", func(c *gin.Context) { controllers.Register(c, dbs) })
		routerAuth.GET("/verifikasi/:uid", func(c *gin.Context) { controllers.Verifikasi(c, dbs) })
		routerAuth.POST("/login", func(c *gin.Context) { login(c, dbs) })
	}
}

type UnsignedResponse struct {
	Message interface{} `json:"message"`
}

func login(c *gin.Context, dbs helper.DBStruct) {
	jsonData, _ := ioutil.ReadAll(c.Request.Body)
	postdata := map[string]interface{}{}
	json.Unmarshal(jsonData, &postdata)

	if cast.ToString(postdata["username"]) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Username cannot be empty",
		})
		return
	} else if cast.ToString(postdata["password"]) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Password cannot be empty",
		})
		return
	}

	username := postdata["username"]
	pasword := helper.Pass2Hash(cast.ToString(postdata["password"]))

	sqlCheckUser := `SELECT * FROM m_users WHERE username='` + cast.ToString(username) + `' AND other_data->>'is_active'='1'`
	rowCheckUser := dbs.DatabaseQuerySingleRow(sqlCheckUser)

	if len(rowCheckUser) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Username not registered",
		})
		return
	} else if pasword != cast.ToString(rowCheckUser["password"]) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Username or Password wrong",
		})
		return
	}

	// CreateJWTToken create token
	JWTTokenSecret := viper.GetString("jwt.token_secret")
	JWTExp := viper.GetInt("jwt.expaired_duration")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"level":    rowCheckUser["level"],
		"user_id":  rowCheckUser["user_id"],
		"exp":      time.Now().Add(time.Second * time.Duration(JWTExp)).Unix(),
	})

	tokenStr, err := token.SignedString([]byte(JWTTokenSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, UnsignedResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"jwt": tokenStr,
		},
		"message": "Sukses Login",
	})
}

func extractBearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("bad header value given")
	}

	jwtToken := strings.Split(header, " ")
	if len(jwtToken) != 2 {
		return "", errors.New("incorrectly formatted authorization header")
	}

	return jwtToken[1], nil
}

func jwtTokenCheck(c *gin.Context) {
	jwtToken, err := extractBearerToken(c.GetHeader("Authorization"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
			Message: err.Error(),
		})
		return
	}

	token, err := parseToken(jwtToken)
	// fmt.Println("token:", token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
			Message: "bad jwt token",
		})
		return
	}

	_, OK := token.Claims.(jwt.MapClaims)
	if !OK {
		c.AbortWithStatusJSON(http.StatusInternalServerError, UnsignedResponse{
			Message: "unable to parse claims",
		})
		return
	}
	c.Next()
}

func parseToken(jwtToken string) (*jwt.Token, error) {
	JWTTokenSecret := viper.GetString("jwt.token_secret")
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		if _, OK := token.Method.(*jwt.SigningMethodHMAC); !OK {
			return nil, errors.New("bad signed method received")
		}
		return []byte(JWTTokenSecret), nil
	})

	if err != nil {
		fmt.Println("error", err.Error())
		return nil, errors.New("bad jwt token")
	}

	return token, nil
}

func privateACLCheck(c *gin.Context) {
	jwtToken, err := extractBearerToken(c.GetHeader("Authorization"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
			Message: err.Error(),
		})
		return
	}

	token, err := parseToken(jwtToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
			Message: "bad jwt token",
		})
		return
	}

	claims, OK := token.Claims.(jwt.MapClaims)
	if !OK {
		c.AbortWithStatusJSON(http.StatusInternalServerError, UnsignedResponse{
			Message: "unable to parse claims",
		})
		return
	}

	// fmt.Println("claims",claims)

	claimedUID, OK := claims["level"]
	if !OK {
		c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
			Message: "no user property in claims",
		})
		return
	}

	// uid := c.Param("uid")
	// if claimedUID != uid {
	// 	c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
	// 		Message: "token uid does not match resource uid",
	// 	})
	// 	return
	// }

	if cast.ToInt(claimedUID) != 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
			Message: "sorry, in this module you do not have access privileges",
		})
		return
	}

	c.Next()
}

func privatePenggunaCheck(c *gin.Context) {
	jwtToken, err := extractBearerToken(c.GetHeader("Authorization"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
			Message: err.Error(),
		})
		return
	}

	token, err := parseToken(jwtToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
			Message: "bad jwt token",
		})
		return
	}

	claims, OK := token.Claims.(jwt.MapClaims)
	if !OK {
		c.AbortWithStatusJSON(http.StatusInternalServerError, UnsignedResponse{
			Message: "unable to parse claims",
		})
		return
	}

	// fmt.Println("claims",claims)

	claimedUID, OK := claims["level"]
	if !OK {
		c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
			Message: "no user property in claims",
		})
		return
	}

	// uid := c.Param("uid")
	// if claimedUID != uid {
	// 	c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
	// 		Message: "token uid does not match resource uid",
	// 	})
	// 	return
	// }

	if cast.ToInt(claimedUID) != 2 {
		c.AbortWithStatusJSON(http.StatusBadRequest, UnsignedResponse{
			Message: "sorry, in this module you do not have access privileges",
		})
		return
	}

	c.Next()
}
