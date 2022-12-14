package router

import (
	"broker-service/user-proto/users"
	"context"
	"helper"
	"io"
	"net/http"
	"time"
	"user-service/models"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewRouter() *gin.Engine {
	routes := gin.Default()
	routes.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "broker")
	})

	routes.GET("/users", func(ctx *gin.Context) {
		var res helper.JsonResponse
		conn, err := grpc.Dial("user-service:8081", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			res.Error = true
			res.Message = err.Error()
			ctx.JSON(http.StatusBadRequest, res)
			return
		}
		defer conn.Close()

		c := users.NewUserServiceClient(conn)
		ctxx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		stream, err := c.FindUsers(ctxx, &emptypb.Empty{})
		if err != nil {
			res.Error = true
			res.Message = err.Error()
			ctx.JSON(http.StatusBadRequest, res)
			return
		}
		var data []*users.User
		for {
			result, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				res.Error = true
				res.Message = err.Error()
				ctx.JSON(http.StatusBadRequest, res)
				return
			}
			data = append(data, result.GetUser())
		}

		res.Error = false
		res.Data = data
		ctx.JSON(http.StatusCreated, res)
	})

	routes.POST("/user/create", func(ctx *gin.Context) {
		var payload models.UserPayload
		var res helper.JsonResponse
		err := ctx.ShouldBindJSON(&payload)
		if err != nil {
			res.Error = true
			res.Message = err.Error()
			ctx.JSON(http.StatusBadRequest, res)
			return
		}

		bio := users.User{
			Fname:    payload.Fname,
			Lname:    payload.Lname,
			Username: payload.Username,
			Email:    payload.Email,
		}

		newUser := users.UserPayload{
			Password: payload.Password,
			User:     &bio,
		}

		conn, err := grpc.Dial("user-service:8081", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			res.Error = true
			res.Message = err.Error()
			ctx.JSON(http.StatusBadRequest, res)
			return
		}
		defer conn.Close()

		c := users.NewUserServiceClient(conn)
		ctxx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		result, err := c.CreateUser(ctxx, &newUser)
		if err != nil {
			res.Error = true
			res.Message = err.Error()
			ctx.JSON(http.StatusBadRequest, res)
			return
		}

		res.Error = false
		res.Message = "NEWLY USER CREATED"
		res.Data = result.GetId()
		ctx.JSON(http.StatusCreated, res)
	})

	return routes
}
