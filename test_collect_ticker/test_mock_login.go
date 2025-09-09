// 模拟登录接口测试程序
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// 模拟登录响应结构
type MockLoginResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
}

// 模拟登录请求结构
type MockLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

func main() {
	// 创建模拟登录接口
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 解析请求
		var loginReq MockLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// 模拟验证逻辑
		if loginReq.Username == "gwatch" && loginReq.Password == "gwatch@vms2025" {
			// 模拟成功登录
			response := MockLoginResponse{
				Code: 200,
				Msg:  "登录成功",
				Data: struct {
					Token string `json:"token"`
				}{
					Token: "mock_token_" + fmt.Sprintf("%d", time.Now().Unix()),
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			log.Printf("模拟登录成功: username=%s, token=%s", loginReq.Username, response.Data.Token)
		} else {
			// 模拟登录失败
			response := MockLoginResponse{
				Code: 401,
				Msg:  "用户名或密码错误",
				Data: struct {
					Token string `json:"token"`
				}{},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			log.Printf("模拟登录失败: username=%s", loginReq.Username)
		}
	})

	// 创建模拟设备接口
	http.HandleFunc("/api/device/query/getDDCTree", func(w http.ResponseWriter, r *http.Request) {
		// 检查Authorization头
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// 模拟设备数据响应
		response := map[string]interface{}{
			"code": 200,
			"msg":  "success",
			"data": []map[string]interface{}{
				{
					"channelOnLineNumber":  85,
					"channelOffLineNumber": 15,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		log.Printf("模拟设备接口调用成功: auth=%s", auth)
	})

	fmt.Println("模拟服务器启动在 :8080")
	fmt.Println("登录接口: http://localhost:8080/login")
	fmt.Println("设备接口: http://localhost:8080/api/device/query/getDDCTree")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
