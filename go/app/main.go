package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	_ "github.com/mattn/go-sqlite3"
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
	dbPath   = "/Users/fukagihina/mercari-build-training/db/mercari.sqlite3"
)

type Response struct {
	Message string `json:"message"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func getItems(c echo.Context) error {
	//ファイルを開く
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		c.Logger().Errorf("Error opening file: %s", err)
		res := Response{Message: "Error opening file"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	defer db.Close()
	//🟥
	cmd := "SELECT items.name, category.name as category, items.image_name FROM items join category on items.category_id = category.id;"
	rows, err := db.Query(cmd)
	if err != nil {
		c.Logger().Errorf("Error getItems Query: %s", err)
		res := Response{Message: "Error getItems Query"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	defer rows.Close()

	items := new(Items)

	for rows.Next() {
		var itemData Item

		err := rows.Scan(&itemData.ID, &itemData.Category, &itemData.ImageName)
		if err != nil {
			c.Logger().Errorf("Error Scan: %s", err)
			res := Response{Message: "Error Scan itemData"}
			return echo.NewHTTPError(http.StatusInternalServerError, res)
		}
		items.Items = append(items.Items, itemData)
	}
	//json形式に変換
	return c.JSON(http.StatusOK, items)
}

func getItemById(c echo.Context) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		c.Logger().Errorf("Error opening file: %s", err)
		res := Response{Message: "Error opening file"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	defer db.Close()
	//idを取得
	id := c.Param("id")
	itemID, err := strconv.Atoi(id)
	if err != nil {
		res := Response{Message: "Error geting itemID"}
		return c.JSON(http.StatusInternalServerError, res)
	}
	var item Item

	query := "SELECT items.name, category.name as category, items.image_name FROM items join category on items.category_id = category.id WHERE items.id = ?"
	row := db.QueryRow(query, itemID)
	err = row.Scan(&item.Name, &item.Category, &item.ImageName)
	if err != nil {
		c.Logger().Errorf("Error Query: %s", err)
		res := Response{Message: "Error Query"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	return c.JSON(http.StatusOK, item)
}

// イメージファイルのハッシュを作成する
func makeHashImage(c echo.Context, image string) (string, error) {
	imageFile, err := c.FormFile("image")
	if err != nil {
		return "", fmt.Errorf("imageFileError: %w", err)
	}
	imageData, err := imageFile.Open()
	if err != nil {
		return "", fmt.Errorf("imageDataError: %w", err)
	}
	defer imageData.Close()
	//ハッシュ値を生成
	hash := sha256.New()
	if _, err := io.Copy(hash, imageData); err != nil {
		return "", fmt.Errorf("HashError: %w", err)
	}
	// バイトのスライスとして、最終的なハッシュ値を得る
	bs := hash.Sum(nil)
	fmt.Printf("%x\n", bs)
	//import encoding/hex: 16 進エンコーディングして返す！
	return hex.EncodeToString(bs), nil
}

// Handler
// func addItem(c echo.Context) error {
// 	var items Items
// 	name := c.FormValue("name")
// 	category := c.FormValue("category")
// 	image, err := c.FormFile("image")
// 	if err != nil {
// 		return err
// 	}

// 	imageHash, err := makeHashImage(c, image.Filename)
// 	if err != nil {
// 		return err
// 	}
// 	imageName := imageHash + ".jpg"

// 	item := Item{Name: name, Category: category, ImageName: imageName}
// 	items.Items = append(items.Items, item)

// 	//db接続
// 	db, err := sql.Open("sqlite3", dbPath)
// 	if err != nil {
// 		c.Logger().Errorf("Error opening file: %s", err)
// 		res := Response{Message: "Error opening file"}
// 		return echo.NewHTTPError(http.StatusInternalServerError, res)
// 	}
// 	defer db.Close()

// 	// カテゴリが存在するか調べる
// 	var categoryID int
// 	row := db.QueryRow("SELECT id FROM category WHERE name = $1", item.Category)
// 	err = row.Scan(&categoryID)
// 	// カテゴリが存在しない場合、新しいカテゴリを追加
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			_, err = db.Exec("INSERT INTO categories (name) VALUES ($1)", item.Category)
// 			if err != nil {
// 				return err
// 			}
// 			row := db.QueryRow("SELECT id FROM category WHERE name = $1", item.Category)
// 			err = row.Scan(&categoryID)
// 			if err != nil {
// 				return err
// 			}
// 		} else {
// 			return err
// 		}
// 	}
// 	cmd2 := "INSERT INTO items (name, category_id, image_name) VALUES ($1, $2, $3)"
// 	_, err = db.Exec(cmd2, item.Name, categoryID, item.ImageName)
// 	if err != nil {
// 		return err
// 	}
// 	message := fmt.Sprintf("item received: name=%s,category=%s,images=%s", name, category, imageName)
// 	res := Response{Message: message}
// 	return c.JSON(http.StatusOK, res)
// }

func addItem(c echo.Context) error {
	// var items Items
	// var categoryID int
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
	imageName := imageHash + ".jpg"

	//db接続
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		c.Logger().Errorf("Error opening file: %s", err)
		res := Response{Message: "Error opening file"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	defer db.Close()

	// カテゴリが存在するか調べる
	var categoryID int64

	err = db.QueryRow("SELECT id FROM category WHERE name = ?", category).Scan(&categoryID)
	// カテゴリが存在しない場合、新しいカテゴリを追加
	if err == sql.ErrNoRows {
		result, err := db.Exec("INSERT INTO category (name) VALUES (?)", category)
		if err != nil {
			res := Response{Message: "Error adding new category to the database"}
			return c.JSON(http.StatusInternalServerError, res)
		}
		categoryID, _ = result.LastInsertId()
	} else if err != nil {
		res := Response{Message: "Error querying category from the database"}
		return c.JSON(http.StatusInternalServerError, res)
	}
	// dbに保存
	stmt, err := db.Prepare("INSERT INTO items (name, category_id, image_name) VALUES (?, ?, ?)")
	if err != nil {
		c.Logger().Errorf("Error INSERT INTO items: %s", err)
		res := Response{Message: "Error INSERT INTO items"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	defer stmt.Close()
	//stmtを元に結果を返す。
	if _, err = stmt.Exec(name, categoryID, imageName); err != nil {
		c.Logger().Errorf("Error opening file: %s", err)
		res := Response{Message: "Error opening file"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	message := fmt.Sprintf("item received: name=%s,category=%s,images=%s", name, category, imageName)
	res := Response{Message: message}
	return c.JSON(http.StatusOK, res)
}

func searchItem(c echo.Context) error {

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer db.Close()

	keyword := c.QueryParam("keyword")
	rows, err := db.Query("SELECT name, category, image_name FROM items WHERE name LIKE ?", "%"+keyword+"%")
	if err != nil {
		c.Logger().Errorf("Error SELECT item: %s", err)
		res := Response{Message: "Error SELECT item"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	defer rows.Close()

	items := new(Items)
	for rows.Next() {
		var itemData Item

		err := rows.Scan(&itemData.Name, &itemData.Category, &itemData.ImageName)
		if err != nil {
			c.Logger().Errorf("Error Scan: %s", err)
			res := Response{Message: "Error Scan itemData"}
			return echo.NewHTTPError(http.StatusInternalServerError, res)
		}
		items.Items = append(items.Items, itemData)
	}
	//json形式に変換
	return c.JSON(http.StatusOK, items)
}

// func addItem(c echo.Context) error {
// 	name := c.FormValue("name")
// 	category := c.FormValue("category")
// 	image, err := c.FormFile("image")
// 	if err != nil {
// 		return err
// 	}

// 	imageHash, err := makeHashImage(c, image.Filename)
// 	if err != nil {
// 		return err
// 	}

// 	newItem := Item{Name: name, Category: category, ImageName: imageHash + ".jpg"}

// 	// Read existing items from JSON file
// 	items, err := readItems()
// 	if err != nil {
// 		c.Logger().Errorf("Error geting hash: %s", err)
// 		return err
// 	}
// 	// Append new item to items
// 	items.Items = append(items.Items, newItem)
// 	// Write items back to JSON file
// 	if err := writeItems(items); err != nil {
// 		return err
// 	}
// 	message := fmt.Sprintf("Item received: %s, category: %s, image: %s", newItem.Name, newItem.Category, newItem.ImageName)
// 	res := Response{Message: message}

// 	return c.JSON(http.StatusOK, res)
// }

// Handler
// func getItems(c echo.Context) error {
// 	items, err := readItems()
// 	if err != nil {
// 		return err
// 	}
// 	return c.JSON(http.StatusOK, items)
// }

// Handler
func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Error image path"}
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
	e.GET("/search", searchItem)
	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
