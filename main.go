/*

   Copyright 2019 MapR Technologies

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.

*/

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	client "github.com/mapr/maprdb-go-client"
)

func buildConnectionString(maprURL *string, auth *string, user *string, password *string, ssl *bool) string {
	var connectionString = *maprURL
	if auth != nil {
		connectionString += fmt.Sprintf("?%s=%s", "auth", *auth)
	}
	if user != nil {
		connectionString += fmt.Sprintf(";%s=%s", "user", *user)
	}
	if password != nil && len(*password) > 0 {
		connectionString += fmt.Sprintf(";%s=%s", "password", *password)
	} else {
		fmt.Println("Password for connection to MapR needs to be set")
		return ""
	}
	if ssl != nil {
		connectionString += fmt.Sprintf(";%s=%t", "ssl", *ssl)
	}
	return connectionString
}

func connectMapR(connectionString string, storeName string) (*client.Connection, *client.DocumentStore) {

	var store *client.DocumentStore
	connection, err := client.MakeConnection(connectionString)
	if err != nil {
		panic(err)
	}
	if exists, _ := connection.IsStoreExists(storeName); exists == false {
		fmt.Printf("Creating store: %s\n", storeName)
		store, err = connection.CreateStore(storeName)
	} else {
		fmt.Printf("Get store: %s\n", storeName)
		store, err = connection.GetStore(storeName)
	}
	if err != nil {
		panic(err)
	}
	return connection, store
}

func storeInDB(conn *client.Connection, store *client.DocumentStore, messages chan []byte) {
	for {
		select {
		case msg := <-messages:
			fmt.Println("Got new message to store in db")
			document, err := conn.CreateDocumentFromString(string(msg))
			if err != nil {
				fmt.Println("Connection error")
				panic(err)
			}
			document.SetIdString(fmt.Sprintf("%s", time.Now()))
			err = store.InsertDocument(document)
			if err != nil {
				fmt.Println("Insert document error")
				panic(err)
			}
			fmt.Println("Inserted document...")

		default:
			fmt.Println("waiting for messages on channel...")
		}
		time.Sleep(1 * time.Second)
	}
}

func createMQTTClientOptions(clientID string, uri *url.URL) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetClientID(clientID)
	return opts
}

func connectMQTT(clientID string, uri *url.URL) mqtt.Client {
	opts := createMQTTClientOptions(clientID, uri)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
	return client
}

func readMQTT(uri *url.URL, topic string, messages chan []byte) {
	client := connectMQTT("sub", uri)
	client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		messages <- msg.Payload()
		fmt.Println("Read message from MQTT")
	})

}

func main() {
	mqttURL := flag.String("mqtt-url", "localhost:1883", "The URL to the MQTT broker")
	maprURL := flag.String("mapr-url", "localhost:5678", "The URL to mapr in the form localhost:5678")
	auth := flag.String("auth", "basic", "Authorization type")
	user := flag.String("user", "mapr", "Username for the connection")
	password := flag.String("password", "", "Password for the user")
	ssl := flag.Bool("use-ssl", false, "Use SSL? (true)")
	storeName := flag.String("mapr-tablename", "/demo/tables/sensor_data", "Table to store the MQTT messages")
	mqttTopic := flag.String("mqtt-topic", "mac/Processes", "MQTT Topic to read from")
	flag.Parse()

	address, _ := url.Parse("tcp://" + *mqttURL)
	connectionString := buildConnectionString(maprURL, auth, user, password, ssl)
	messages := make(chan []byte)

	fmt.Println("Formatted connection string to MapR:")
	fmt.Println(connectionString)

	/* Application starts here */
	connection, store := connectMapR(connectionString, *storeName)
	go readMQTT(address, *mqttTopic, messages)
	go storeInDB(connection, store, messages)
	/* wait until we see a eot (^D) on stdin */
	readval, _ := ioutil.ReadAll(os.Stdin)
	fmt.Println(readval)
	connection.Close()
}
