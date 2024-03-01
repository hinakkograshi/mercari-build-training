package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
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
			c.Logger().Errorf("Error Scan item: %s", err)
			res := Response{Message: "Error Scan item"}
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
	query := "SELECT items.id, items.name, categories.name as categories, items.image_name FROM items join categories on items.category_id = categories.id WHERE items.id = ?"
	row := db.QueryRow(query, itemID)
	err = row.Scan(&item.ID, &item.Name, &item.Category, &item.ImageName)
	if err != nil {
		c.Logger().Errorf("Error Query: %s", err)
		res := Response{Message: "Error Query"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	return c.JSON(http.StatusOK, item)
}

// イメージファイルのハッシュ化
func saveImage(file *multipart.FileHeader) (string, error) {

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	// ハッシュ計算
	hash := sha256.New()
	if _, err := io.Copy(hash, src); err != nil {
		return "", err
	}
	// ハッシュを16進数文字列に変換
	hashString := hash.Sum(nil)
	// 行先ファイルを作成
	hashedImageName := fmt.Sprintf("%x.jpg", hashString)
	dst, err := os.Create(path.Join(ImgDir, hashedImageName))
	if err != nil {
		return "", err
	}
	defer dst.Close()
	// 行先ファイルに保存
	src.Seek(0, 0)
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return hashedImageName, nil
}

func addItem(c echo.Context) error {
	name := c.FormValue("name")
	category := c.FormValue("category")
	image, err := c.FormFile("image")
	if err != nil {
		res := Response{Message: "Return image FormFile"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}

	imageName, err := saveImage(image)
	if err != nil {
		res := Response{Message: "Return imageHash"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
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
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		c.Logger().Errorf("Error opening file: %s", err)
		res := Response{Message: "Error opening file"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	defer db.Close()
	// id+.jpg
	imageFilename := c.Param("imageFilename")
	imageJpg := imageFilename + ".jpg"
	fmt.Println("imageFilename!!!!!!!!:%v:", imageJpg)

	imgPath := path.Join(ImgDir, imageJpg)

	//拡張子がjpgがチェック
	if !strings.HasSuffix(imgPath, ".jpg") {
		c.Logger().Errorf("Image path does not end with .jpg:%s", imgPath)
		return echo.NewHTTPError(http.StatusInternalServerError, "Image path does not end with .jpg")
	}
	// .jpgを取り除く
	imageID := strings.TrimSuffix(imageJpg, ".jpg")
	var imgPathById string
	row := db.QueryRow("SELECT image_name FROM items WHERE id = ?", imageID)

	err = row.Scan(&imgPathById)
	if err != nil {
		c.Logger().Errorf("Error Scan item: %s", err)
		res := Response{Message: "Error Scan item"}
		return echo.NewHTTPError(http.StatusInternalServerError, res)
	}
	fmt.Println("row!!!!!!!!:%v:", row)
	fmt.Println("imgPathById!!!!!!!!:%v:", imgPathById)
	fmt.Println("imgPath!!!!!!!!:%v:", imgPath)

	imgPath = path.Join(ImgDir, imgPathById)

	fmt.Println("imgPath!??????!!:%v:", imgPath)

	// ファイルが存在しないときはdefault.jpgを表示
	if _, err := os.Stat(imgPath); err != nil {
		fmt.Println("imgPath?????????!!!!!!!!:%v:", imgPath)
		c.Logger().Errorf("Image not found: %s", err)
		fmt.Print("Image not found: %s", err)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	fmt.Println("imgPath#########:%v:", imgPath)
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
