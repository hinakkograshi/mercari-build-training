package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

type Items struct {
	Items []Item `json:"item"`
}

type Item struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	ImageName string `json:"image_name"`
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
	jsonItemData, err := os.ReadFile(JSONFile)
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

func getItemById(c echo.Context) error {
	//itemID取得
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Logger().Errorf("Error geting item id: %s", err)
	}
	//ファイルを開く
	file, err := os.Open(JSONFile)
	if err != nil {
		c.Logger().Errorf("Error opening file: %s", err)
		res := Response{Message: "Error opening file"}
		return c.JSON(http.StatusInternalServerError, res)
	}
	defer file.Close()

	var itemsData Items
	err = json.NewDecoder(file).Decode(&itemsData)
	if err != nil {
		c.Logger().Errorf("Error decoding file: %s", err)
		res := Response{Message: "Error decoding file"}
		return c.JSON(http.StatusInternalServerError, res)
	}
	//id-1が0未満ならエラー
	indexID := id - 1
	if indexID < 0 || indexID > len(itemsData.Items)-1 {
		return err
	}

	return c.JSON(http.StatusOK, itemsData.Items[indexID])
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

// イメージファイルのハッシュを作成する
func makeHashImage(c echo.Context, image string) (string, error) {
	imageFile, err := c.FormFile("image")
	if err != nil {
		return "", fmt.Errorf("imageFileError")
	}
	imageData, err := imageFile.Open()
	if err != nil {
		return "", fmt.Errorf("imageDataError")
	}
	defer imageData.Close()
	//ハッシュ値を生成
	hash := sha256.New()
	if _, err := io.Copy(hash, imageData); err != nil {
		return "", fmt.Errorf("HashError")
	}
	// バイトのスライスとして、最終的なハッシュ値を得る
	bs := hash.Sum(nil)
	fmt.Printf("%x\n", bs)
	//import encoding/hex: 16 進エンコーディングして返す！
	return hex.EncodeToString(bs), nil
}

// Handler
func addItem(c echo.Context) error {
	name := c.FormValue("name")
	category := c.FormValue("category")
	image, err := c.FormFile("image")
	if err != nil {
		return err
	}

	imageHash, err := makeHashImage(c, image.Filename)
	if err != nil {
		return err
	}

	newItem := Item{Name: name, Category: category, ImageName: imageHash + ".jpg"}

	// Read existing items from JSON file
	items, err := readItems()
	if err != nil {
		c.Logger().Errorf("Error geting hash: %s", err)
		return err
	}
	// Append new item to items
	items.Items = append(items.Items, newItem)
	// Write items back to JSON file
	if err := writeItems(items); err != nil {
		return err
	}
	message := fmt.Sprintf("Item received: %s, category: %s, image: %s", newItem.Name, newItem.Category, newItem.ImageName)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
}

// Handler
func getItems(c echo.Context) error {
	items, err := readItems()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, items)
}

// Handler
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
	//echoインスタンス生成
	e := echo.New()
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Start server
	e.Logger.SetLevel(log.DEBUG)

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
	e.GET("/items/:id", getItemById)
	e.GET("/image/:imageFilename", getImg)
	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
