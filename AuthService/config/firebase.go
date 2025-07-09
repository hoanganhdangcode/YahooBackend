package config

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

var FirestoreClient *firestore.Client

func ConnectFirebase() {
	ctx := context.Background()

	opt := option.WithCredentialsFile("../ooo-6027e-firebase-adminsdk-fbsvc-f93a80c12a.json")

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Panicf("init app error: %v", err)
	}
	client, err := app.Firestore(ctx)
	if err != nil {
		log.Panicf("firestore connect error: %v", err)
	}
	FirestoreClient = client
	log.Println("Kết nối Firestore OK")
}
