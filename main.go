package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"time"
)

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
	catToken := flag.String("token", "", "The Cat API - API key")
	flag.Parse()

	if *catToken == "" {
		panic("No API key provided")
	}

	http.Handle("/totally-not-a-virus", http.RedirectHandler("/cat", http.StatusFound))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%s; %s]: %s\n", time.Now().Format(time.DateTime), r.Host+r.URL.Path, r.RemoteAddr)

		http.ServeFile(w, r, "home.go.html")
	})

	http.HandleFunc("/cat", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%s; %s]: %s\n", time.Now().Format(time.DateTime), r.Host+r.URL.Path, r.RemoteAddr)
		templ := template.Must(template.New("cats.go.html").ParseFiles("cats.go.html"))

		catReq, err := http.NewRequest("get", "https://api.thecatapi.com/v1/images/search?has_breeds=1", nil)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		catReq.Header.Set("x-api-key", *catToken)

		res, err := http.DefaultClient.Do(catReq)
		if err != nil {
			fmt.Println(err)
			return
		}

		catRes, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return
		}

		var catResponse []CatResponse
		if err := json.Unmarshal(catRes, &catResponse); err != nil {
			fmt.Printf("Error on %s. %v", catRes, err)
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
			fmt.Println(err)
			return
		}
	})

	fmt.Println("Starting server!")

	lis, err := net.Listen("tcp6", ":80")
	if err != nil {
		panic(fmt.Sprintf("server: Could not bind to port 80. %v", err))
	}

	err = http.Serve(lis, nil)
	if err != nil {
		panic(fmt.Sprintf("server: Could not start the server. %v", err))
	}
}
