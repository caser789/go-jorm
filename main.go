package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"xorm.io/xorm"

	_ "github.com/go-sql-driver/mysql"
)

type Album struct {
	Id int64 `xorm:"id"`
	Title string
	Artist string
	Price float32
}

func officialQueryRow() {
	db, err := sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/recording")
	if err != nil {
		fmt.Printf("open DB error: %v\n", err)
	}
	defer db.Close()

	var album Album

	err = db.QueryRow("SELECT * FROM album WHERE id = ?", 1).Scan(&album.Id, &album.Title, &album.Artist, &album.Price)
	if err != nil {
		fmt.Printf("query row error: %v\n", err)
	}

	fmt.Println(album)
}

func xormQueryRow() {
	orm, err:= xorm.NewEngine("mysql", "root:password@tcp(127.0.0.1:3306)/recording")
	if err != nil {
		fmt.Printf("xorm open DB error: %v\n", err)
	}

	album := Album{Id: 1}

	has, err := orm.Get(&album)
	if err != nil {
		fmt.Printf("xorm GET error: %v\n", err)
	}
	if !has {
		fmt.Printf("xorm GET doesn't have\n")
	}

	fmt.Println(album)
}

func jormQueryRow() {
	db, err := sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/recording")
	if err != nil {
		fmt.Printf("open DB error: %v\n", err)
	}
	defer db.Close()

	album := &Album{Id: 1}

	beanValue := reflect.ValueOf(album) // beanValue.Kind() should be reflect.Ptr
	beanElem := beanValue.Elem()
	beanType := beanElem.Type()
	tableName := beanElem.Type().Name()
	tableName = snakeCaseName(tableName)

	var cols []string
	var conditions []string
	var args []interface{}
	dbName2StructName := map[string]string{}

	for i := 0; i < beanType.NumField(); i++ {
		col := beanType.Field(i).Name
		cols = append(cols, snakeCaseName(col))
		dbName2StructName[snakeCaseName(col)] = col

		val := reflect.Indirect(beanValue).FieldByName(col)
		if !val.IsZero() {
			conditions = append(conditions, snakeCaseName(col))
			args = append(args, val.Interface())
		}
	}

	resultClause := strings.Join(cols, ", ")

	var conditionWithPlaceholder []string
	for _, s := range conditions {
		conditionWithPlaceholder = append(conditionWithPlaceholder, fmt.Sprintf(" %s = ? ", s))
	}
	conditionClause := strings.Join(conditionWithPlaceholder, " AND ")

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s ;", resultClause, tableName, conditionClause)
	// fmt.Printf("query is %s %v \n", query, args)

	scanResults := make([]interface{}, beanType.NumField())
	for i := 0; i < beanType.NumField(); i++ {
		var cell interface{}
		scanResults[i] = &cell
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Println("jorm query error %v\n", err)
	}
	defer rows.Close()

	cols, err = rows.Columns()
	if err != nil {
		fmt.Println("jorm Columns error %v\n", err)
	}
	var resultsSlice []map[string][]byte
	for rows.Next() {
		var scanResultContainers []interface{}
		for i := 0; i < len(cols); i++ {
			var scanResultContainer interface{}
			scanResultContainers = append(scanResultContainers, &scanResultContainer)
		}

		if err := rows.Scan(scanResultContainers...); err != nil {
			fmt.Printf("rows.Scan error %v\n", err)
		}

		result := make(map[string][]byte)
		for ii, key := range cols {
			rawValue := reflect.Indirect(reflect.ValueOf(scanResultContainers[ii]))
			aa := reflect.TypeOf(rawValue.Interface())
			vv := reflect.ValueOf(rawValue.Interface())
			var str string
			switch aa.Kind() {
			case reflect.Int64:
				str = strconv.FormatInt(vv.Int(), 10)
				result[key] = []byte(str)
			case reflect.Slice:
				if aa.Elem().Kind() == reflect.Uint8 {
					result[key] = rawValue.Interface().([]byte)
				}
			}
		}
		resultsSlice = append(resultsSlice, result)
	}

	result := resultsSlice[0]

	albumStruct := reflect.Indirect(reflect.ValueOf(album))
	for key, data := range result {
		structField := albumStruct.FieldByName(dbName2StructName[key])
		var v interface{}
		switch structField.Type().Kind() {
		case reflect.Int64:
			x, err := strconv.ParseInt(string(data), 10, 64)
			if err != nil {}
			v = x
		case reflect.String:
			v = string(data)
		case reflect.Float32:
			x, err := strconv.ParseFloat(string(data), 64)
			if err != nil {}
			v = float32(x)
		}
		structField.Set(reflect.ValueOf(v))
	}

	fmt.Println(album)
}

func snakeCaseName(name string) string {
	newstr := make([]byte, 0, len(name)+1)
	for i := 0; i < len(name); i++ {
		c := name[i]
		if isUpper := 'A' <= c && c <= 'Z'; isUpper {
			if i > 0 {
				newstr = append(newstr, '_')
			}
			c += 'a' - 'A'
		}
		newstr = append(newstr, c)
	}
	return b2s(newstr)
}

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func main() {
	// officialQueryRow()
	// xormQueryRow()
	jormQueryRow()
}