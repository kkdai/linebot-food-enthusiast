package main

import (
	"context"
	"log"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/db"
	"google.golang.org/api/option"
)

// Food is the struct for the food data
type Food struct {
	Name     string `json:"name"`
	Calories int    `json:"calories"`
	Time     string `json:"time"`
}

// DBFoodPath is the path to the namecard data in the database
const DBFoodPath = "food"

// Define the context
var fireDB FireDB

// define firebase db
type FireDB struct {
	path string
	ctx  context.Context
	*db.Client
}

// SetPath sets the path of the location in the database
func (f *FireDB) SetPath(path string) {
	f.path = path
}

// GetRef returns a reference to the location at the specified path.
func (f *FireDB) GetFromDB(data interface{}) error {
	if err := f.NewRef(f.path).Get(f.ctx, data); err != nil {
		return err
	}
	return nil
}

// Insert data to firebase
func (f *FireDB) InsertDB(data interface{}) error {
	_, err := f.NewRef(f.path).Push(f.ctx, data)
	if err != nil {
		return err
	}
	return nil
}

// initFirebase: Initialize firebase
func initFirebase(gap, firebaseURL string, ctx context.Context) {
	log.Println("initFirebase")

	opt := option.WithCredentialsJSON([]byte(gap))
	config := &firebase.Config{DatabaseURL: firebaseURL}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		log.Fatalf("error initializing firebase app: %v", err)
	}
	client, err := app.Database(ctx)
	if err != nil {
		log.Fatalf("error initializing database: %v", err)
	}
	fireDB.Client = client
	fireDB.ctx = ctx
}
