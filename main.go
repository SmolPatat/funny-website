package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/BurntSushi/toml"
	"github.com/hashicorp/vault-client-go"
)

type Config struct {
	Port  int `toml:"port"`
	Vault struct {
		Address string `toml:"address"`
	} `toml:"vault"`
}

type Breed struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CatResponse struct {
	Id        string  `json:"id"`
	Url       string  `json:"url"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Breeds    []Breed `json:"breeds"`
	Favourite any     `json:"favourite"`
}

type Cat struct {
	ImageURL    string
	Name        string
	Description string
}

func main() {
	var config Config
	_, err := toml.DecodeFile("config.toml", &config)
	if err != nil {
		log.Fatalf("cannot read configuration: %v", err)
	}

	vault, err := vault.New(vault.WithAddress(config.Vault.Address), vault.WithEnvironment())
	if err != nil {
		log.Fatalf("cannot connect to Vault: %v", err)
	}

	catToken, err := getCatToken(vault)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/totally-not-a-virus", http.RedirectHandler("/cat", http.StatusFound))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s by %s\n", r.Host+r.URL.Path, r.RemoteAddr)

		http.ServeFile(w, r, "home.go.html")
	})

	http.HandleFunc("/cat", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s by %s\n", r.Host+r.URL.Path, r.RemoteAddr)
		templ := template.Must(template.New("cats.go.html").ParseFiles("cats.go.html"))

		catReq, err := http.NewRequest("get", "https://api.thecatapi.com/v1/images/search?has_breeds=1", nil)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		catReq.Header.Set("x-api-key", catToken)

		res, err := http.DefaultClient.Do(catReq)
		if err != nil {
			log.Println(err)
			return
		}

		catRes, err := io.ReadAll(res.Body)
		if err != nil {
			log.Println(err)
			return
		}

		var catResponse []CatResponse
		if err := json.Unmarshal(catRes, &catResponse); err != nil {
			log.Printf("Error on %s. %v", catRes, err)
			catResponse = []CatResponse{{
				Breeds: []Breed{{}},
			}}
		}

		cat := Cat{
			Name:        catResponse[0].Breeds[0].Name,
			Description: catResponse[0].Breeds[0].Description,
			ImageURL:    catResponse[0].Url,
		}

		if err := templ.Execute(w, cat); err != nil {
			log.Println(err)
			return
		}
	})

	lis, err := net.Listen("tcp6", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		log.Fatalf("server: Could not bind to port 80. %v", err)
	}

	log.Println("Starting the server!")

	err = http.Serve(lis, nil)
	if err != nil {
		log.Fatalf("server: Could not start the server. %v", err)
	}
}

func getCatToken(client *vault.Client) (string, error) {
	ctx := context.TODO()
	response, err := client.Secrets.KvV2Read(ctx, "the-cat-api", vault.WithMountPath("secret"))
	if err != nil {
		return "", fmt.Errorf("cannot read TheCatAPI secrets: %w", err)
	}

	catToken, ok := response.Data.Data["API-key"].(string)
	if !ok {
		return "", fmt.Errorf("cannot read TheCatAPI token: %w", err)
	}
	return catToken, nil
}
