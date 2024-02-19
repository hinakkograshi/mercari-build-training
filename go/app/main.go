package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

type Items struct {
	Items []Item `json:"item"`
}

type Item struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

const (
	ImgDir   = "images"
	JSONFile = "items.json"
)

type Response struct {
	Message string `json:"message"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func readItems() (*Items, error) {
	jsonItemData, err := os.ReadFile("items.json")
	if err != nil {
		return nil, err
	}
	var addItems Items
	// Decode: JSONからItemsに変換
	if err := json.Unmarshal(jsonItemData, &addItems); err != nil {
		return nil, err
	}
	return &addItems, nil
}

// ItemsからJSONに変換
func writeItems(items *Items) error {
	jsonItemData, err := os.Create(JSONFile)
	if err != nil {
		return err
	}
	defer jsonItemData.Close()
	// Encode: ItemsからJSONに変換

	encoder := json.NewEncoder(jsonItemData)
	if err := encoder.Encode(items); err != nil {
		return err
	}
	return nil
}

func addItem(c echo.Context) error {
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")

	if name == "" || category == "" {
		return c.JSON(http.StatusBadRequest,
			Response{Message: "Name or category cannot be empty"})
	}

	newItem := Item{Name: name, Category: category}
	// for debug: Received item
	fmt.Printf("Received item: %+v\n", newItem)
	// Read existing items from JSON file
	items, err := readItems()
	if err != nil {
		return err
	}
	// Append new item to items
	items.Items = append(items.Items, newItem)
	// Write items back to JSON file
	if err := writeItems(items); err != nil {
		return err
	}
	message := fmt.Sprintf("Item received: %s, category: %s", newItem.Name, newItem.Category)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
}

func getItems(c echo.Context) error {
	items, err := readItems()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, items)
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

	frontURL := os.Getenv("FRONT_URL")
	if frontURL == "" {
		frontURL = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{frontURL},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// Routes
	e.GET("/", root)
	e.POST("/items", addItem)
	e.GET("/items", getItems)
	e.GET("/image/:imageFilename", getImg)
	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
