package main

import (
	"context"
	"log"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/db"
	"google.golang.org/api/option"
)

// Food is the struct for the food data
type Food struct {
	Name     string `json:"name"`
	Calories int    `json:"calories"`
	Date     string `json:"time"`
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

// GetLocalTimeString: Get local time string
func GetLocalTimeString() string {
	timelocal, _ := time.LoadLocation("Asia/Taipei")
	time.Local = timelocal
	curNow := time.Now().Local().String()
	return curNow
}

// recordCalorie: 記錄卡路里攝入
func recordCalorie(foodItem string, date string, calories float64) map[string]any {
	// This hypothetical API returns a JSON such as:
	// {"date":"2024-04-17","calories":200.0,"foodItem":"Apple","status":"Success"}
	calorie := Food{
		Name:     foodItem,
		Date:     date,
		Calories: int(calories),
	}

	// Insert the calorie intake to the database.
	if err := fireDB.InsertDB(calorie); err != nil {
		log.Println("Storage save err:", err)
	}

	return map[string]any{
		"foodItem": foodItem,
		"date":     date,
		"calories": calories,
		"status":   "Success",
	}
}

// listAllCalories: 列出指定日期範圍內的所有卡路里攝入
func listAllCalories(startDate string, endDate string) map[string]any {
	filteredCalories := make(map[string]any)
	// Get all calorie intakes from the database.
	var calories map[string]Food
	if err := fireDB.GetFromDB(&calories); err != nil {
		log.Println("Storage get err:", err)
		return nil
	}

	// Filter the calorie intakes based on the specified date range.
	for _, calorie := range calories {
		calorieDate := calorie.Date
		// Include the calorie intake if it falls within the specified date range or if no dates are specified
		if (startDate == "" && endDate == "") || (calorieDate >= startDate && calorieDate <= endDate) {
			filteredCalories[calorie.Name+"-"+calorie.Date] = calorie
		}
	}
	return filteredCalories
}
