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
	Items []Item `json:"items"`
}

// IDを追加
type Item struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	ImageName string `json:"image_name"`
}

const (
	ImgDir   = "images"
	JSONFile = "items.json"
	dbPath   = "./db/mercari.sqlite3"
)

type Response struct {
	Message string `json:"message"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func getItems(c echo.Context) error {
	//dbに接続
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		c.Logger().Errorf("Error opening file: %s", err)
		res := Response{Message: "Error opening file"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	defer db.Close()
	query := `
	SELECT items.id, items.name, categories.name, items.image_name
	FROM items
	JOIN categories ON items.category_id = categories.id
`
	rows, err := db.Query(query)
	if err != nil {
		c.Logger().Errorf("Error getItems Query: %s", err)
		res := Response{Message: "Error getItems Query"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	defer rows.Close()

	items := new(Items)
	for rows.Next() {
		var item Item
		err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageName)
		if err != nil {
			c.Logger().Errorf("Error Scan itemData: %s", err)
			res := Response{Message: "Error Scan itemData"}
			return echo.NewHTTPError(http.StatusInternalServerError, res)
		}
		items.Items = append(items.Items, item)
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
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	var item Item
	query := "SELECT items.name, categories.name as categories, items.image_name FROM items join categories on items.category_id = categories.id WHERE items.id = ?"
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

func addItem(c echo.Context) error {
	name := c.FormValue("name")
	category := c.FormValue("category")
	image, err := c.FormFile("image")
	if err != nil {
		res := Response{Message: "Return image FormFile"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}

	imageHash, err := makeHashImage(c, image.Filename)
	if err != nil {
		res := Response{Message: "Return imageHash"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
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
	row := db.QueryRow("SELECT id FROM categories WHERE name = ?", category)
	err = row.Scan(&categoryID)
	// カテゴリが存在しない場合、新しいカテゴリを追加
	if err == sql.ErrNoRows {
		result, err := db.Exec("INSERT INTO categories (name) VALUES (?)", category)
		if err != nil {
			res := Response{Message: "Error adding new categories to the database"}
			return echo.NewHTTPError(http.StatusInternalServerError, res)
		}
		categoryID, _ = result.LastInsertId()
	} else if err != nil {
		c.Logger().Errorf("Error INSERT INTO items: %s", err)
		res := Response{Message: "Error querying categories from the database"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
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
	message := fmt.Sprintf("item received: name=%s,categories=%s,images=%s", name, category, imageName)
	res := Response{Message: message}
	return c.JSON(http.StatusOK, res)
}

func searchItem(c echo.Context) error {
	var items Items
	keyword := c.QueryParam("keyword")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		c.Logger().Errorf("Error opening file: %s", err)
		res := Response{Message: "Error opening file"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	defer db.Close()

	query := "SELECT items.name, categories.name, items.image_name FROM items JOIN categories ON items.category_id = categories.id WHERE items.name LIKE ?"
	rows, err := db.Query(query, "%"+keyword+"%")
	if err != nil {
		c.Logger().Errorf("Error Query: %s", err)
		res := Response{Message: "Error Query"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	defer rows.Close()
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.Name, &item.Category, &item.ImageName); err != nil {
			res := Response{Message: "Error Scan earchItem"}
			return echo.NewHTTPError(http.StatusInternalServerError, res)
		}
		items.Items = append(items.Items, item)
	}
	return c.JSON(http.StatusOK, items)
}

// Handler
func getImg(c echo.Context) error {
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Error image path"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
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
