package main

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	HOST        = "elephant.sandyuraz.com"
	PORT        = 5432
	DBNAME      = "sandissa"
	USER        = "sandy"
	SSLMODE     = "verify-full"
	SSLCERT     = "./postgres/client.crt"
	SSLKEY      = "./postgres/client.key"
	SSLROOTCERT = "./postgres/ca.crt"
	tempRange   = 60
)

var (
	DB *gorm.DB

	dsn = fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s",
		HOST, PORT, USER, DBNAME, SSLMODE, SSLCERT, SSLKEY, SSLROOTCERT,
	)
)

// Temperature is the DB model used to store temperatures
type Temperature struct {
	gorm.Model

	Value float64
}

// User is the DB model used to store user data, pass is sha512.
type User struct {
	gorm.Model

	Name     string `gorm:"unique"`
	Password string
}

// init opens the database.
func initDB() error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		lerr("Failed to init storage", err, params{})
		return err
	}
	DB.AutoMigrate(&Temperature{}, &User{})
	if err != nil {
		return err
	}
	return nil
}

// addTempDB logs the temperature value.
func addTempDB(value float64) error {
	return DB.Create(&Temperature{Value: value}).Error
}

// getTempDB returns the last temperature value.
func getTempDB() (*Temperature, error) {
	val := &Temperature{}
	return val, DB.Model(&Temperature{}).Order("ID desc").First(val).Error
}

// getTempsDB returns the last temperature value.
func getTempsDB() ([]Temperature, error) {
	val := make([]Temperature, 0, tempRange)
	err := DB.Model(&Temperature{}).Order("ID desc").Limit(tempRange).Find(&val).Error
	return val, err
}

// addUser creates a user
func addUser(name, pass string) error {
	return DB.Create(&User{Name: name, Password: shaEncode(pass)}).Error
}

// getUser return the user if found.
func getUser(name string) (*User, error) {
	user := &User{}
	return user, DB.Model(&User{}).Where("name = ?", name).First(user).Error
}

// closeDB closes the database.
func closeDB() error {
	db, err := DB.DB()
	if err != nil {
		lerr("Failed to get database while closing", err, params{})
		return err
	}
	err = db.Close()
	if err != nil {
		lerr("Failed to close the database", err, params{})
		return err
	}
	lf("Closed database", params{})
	return nil
}
