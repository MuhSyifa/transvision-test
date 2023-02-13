package helper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	enckey = "go-pertamina-sbm123456789!@#$%^&"
)

// DBStruct ..
type DBStruct struct {
	Dbx *sqlx.DB
}

type Response struct {
	Status  string
	Message string
	Data    string
}

// EncString berfungsi untuk encrypt string, outputnya adalah base64 string
func EncString(plaintext string) string {
	keyinbytes := []byte(enckey)
	plaintextinbytes := []byte(plaintext)
	// fmt.Printf("%s\n", plaintext)
	ciphertext, err := encrypt(keyinbytes, plaintextinbytes)
	if err != nil {
		fmt.Println("Err Encrypt String => ", err)
		return ""
	}

	return base64.StdEncoding.EncodeToString(ciphertext)
}

// DecString berfungsi untuk decrypt string, inputnya adalah base64 string
func DecString(enctext string) string {
	keyinbytes := []byte(enckey)
	encbytes, err1 := base64.StdEncoding.DecodeString(enctext)
	if err1 != nil {
		fmt.Println("Err DecString, Decode Base64 => ", err1)
		return ""
	}
	result, err2 := decrypt(keyinbytes, encbytes)
	if err2 != nil {
		fmt.Println("Err DecString => ", err2)
		return ""
	}
	return string(result)
}

func encrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(text)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return ciphertext, nil
}

func decrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(text) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	data, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return nil, err
	}
	return data, nil
}

//MD5 ..
func MD5(plaintext string) string {
	hasher := md5.New()
	hasher.Write([]byte(plaintext))
	return hex.EncodeToString(hasher.Sum(nil))
}

// DatabaseQueryRows ..
func (dbs *DBStruct) DatabaseQueryRows(query string, args ...interface{}) []map[string]interface{} {

	var datarows []map[string]interface{}
	rows, err := dbs.Dbx.Queryx(query, args...)

	if err != nil {
		fmt.Println("Query Error", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			results := make(map[string]interface{})
			err = rows.MapScan(results)
			if err != nil {
				fmt.Println(err)
			}
			datarows = append(datarows, mapBytesToString(results))
		}
	}

	return datarows
}

// DatabaseQuerySingleRow ..
func (dbs *DBStruct) DatabaseQuerySingleRow(query string, args ...interface{}) map[string]interface{} {

	result := make(map[string]interface{})

	rows, err := dbs.Dbx.Queryx(query, args...)

	if err != nil {
		fmt.Println("Query Error", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			results := make(map[string]interface{})
			err = rows.MapScan(results)
			if err != nil {
				fmt.Println(err)
			}
			return mapBytesToString(results)
		}
	}

	return result
}
func mapBytesToString(m map[string]interface{}) map[string]interface{} {
	for k, v := range m {
		if b, ok := v.([]byte); ok {
			m[k] = string(b)
		}
	}
	return m
}

func ReadFileString(path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("File reading error", err)
		return ""
	}
	return string(data)
}
