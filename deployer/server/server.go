package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	client "github.com/docker/docker/client"

	"github.com/labstack/echo/v4"
)

type Server struct {
	echo   *echo.Echo
	port   int
	client *client.Client
	image  string
}

type User struct {
	Name string `json:"name"`
}

type Url struct {
	Url string `json:"url"`
}

func New() (*Server, error) {
	e := echo.New()
	e.Server.ReadTimeout = 5 * time.Second
	e.Server.WriteTimeout = 30 * time.Second
	e.Server.IdleTimeout = 120 * time.Second
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	return &Server{
		echo:   echo.New(),
		port:   8085,
		client: client,
		image:  "serverless", // <----------------------------------------------------     Change image name here
	}, nil
}

func (s *Server) Run() (err error) {
	s.echo.POST("/enclaves", s.deployment)
	s.echo.DELETE("/enclaves/:id", s.deletion)
	port := "8080" // <----------------------------------------------------     Change server port here
	err = s.echo.Start(":" + port)
	if err == http.ErrServerClosed {
		err = nil
	}
	return
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.echo.Shutdown(ctx)
}

func (s *Server) deployment(c echo.Context) error {
	u := new(User)
	if err := c.Bind(u); err != nil {
		return err
	}
	exists, _ := s.CheckIfServiceExists(c.Request().Context(), u.Name)
	fmt.Printf("service exists: %v", exists)
	port := 0
	if !exists {
		var err error
		port, err = s.DeployContainer(c.Request().Context(), u.Name)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
	}
	return c.JSON(http.StatusOK, Url{Url: fmt.Sprintf("http://127.0.0.1:%v", port)})
}

func (s *Server) deletion(c echo.Context) error {
	username := c.Param("id")
	fmt.Printf("Delete called for user %v \n", username)
	exists, _ := s.CheckIfServiceExists(c.Request().Context(), username)
	fmt.Printf("service exists: %v", exists)
	if exists {
		err := s.DeleteServiceForUser(c.Request().Context(), username)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
	}
	return c.String(http.StatusOK, "")
}
