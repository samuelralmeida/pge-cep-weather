package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/joho/godotenv"
)

type config struct {
	port          string
	weatherApiKey string
}

var conf config

func init() {
	port := os.Getenv("PORT")

	// PORT env é definida pelo cloud run
	// se não encontrar tenta carregar do arquivo .env
	// e então definir a porta que a aplicação vai rodar
	if port == "" {
		err := godotenv.Load()
		if err != nil {
			log.Fatalln("Error loading .env file:", err)
		}
		port = os.Getenv("PORT")
	}

	weatherApiKey := os.Getenv("WEATHER_API_KEY")

	conf = config{
		port:          port,
		weatherApiKey: weatherApiKey,
	}
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(60 * time.Second))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("route does not exist"))
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(405)
		w.Write([]byte("method is not valid"))
	})

	r.Get("/weather/{cep}", weatherHandler)

	http.ListenAndServe(":"+conf.port, r)
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	cep := chi.URLParam(r, "cep")

	if !isValidCep(cep) {
		log.Println("cep inválido:", cep)
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	cepInfo, err := getLocationData(cep)
	if err != nil {
		log.Println("getLocationData:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if cepInfo == nil {
		http.Error(w, "can not find zipcode", http.StatusNotFound)
		return
	}

	weatherInfo, err := getWeatherData(cepInfo.City)
	if err != nil {
		log.Println("getWeatherData:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	render.Status(r, 200)
	render.Render(w, r, weatherInfo)
}

func isValidCep(cep string) bool {
	re := regexp.MustCompile(`^\d{8}$`)
	return re.MatchString(cep)
}

type cepInfo struct {
	Cep          string `json:"cep"`
	State        string `json:"state"`
	City         string `json:"city"`
	Neighborhood string `json:"neighborhood"`
	Street       string `json:"street"`
}

func getLocationData(cep string) (*cepInfo, error) {
	type respData struct {
		Cep         string `json:"cep"`
		Logradouro  string `json:"logradouro"`
		Complemento string `json:"complemento"`
		Unidade     string `json:"unidade"`
		Bairro      string `json:"bairro"`
		Localidade  string `json:"localidade"`
		Uf          string `json:"uf"`
		Ibge        string `json:"ibge"`
		Gia         string `json:"gia"`
		Ddd         string `json:"ddd"`
		Siafi       string `json:"siafi"`
		Err         bool   `json:"erro"`
	}

	var data respData

	err := request(context.Background(), fmt.Sprintf("http://viacep.com.br/ws/%s/json/", cep), &data)
	if err != nil {
		return nil, fmt.Errorf("error requesting via cep: %w", err)
	}

	if data.Err {
		return nil, nil
	}

	resp := &cepInfo{
		Cep:          data.Cep,
		State:        data.Uf,
		City:         data.Localidade,
		Neighborhood: data.Bairro,
		Street:       data.Logradouro,
	}

	return resp, err
}

type WeatherInfo struct {
	Kelvin     float64 `json:"temp_K"`
	Celsius    float64 `json:"temp_C"`
	Fahrenheit float64 `json:"temp_F"`
}

func (wi *WeatherInfo) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func getWeatherData(location string) (*WeatherInfo, error) {
	type respData struct {
		Current struct {
			TempC float64 `json:"temp_c"`
			TempF float64 `json:"temp_f"`
		} `json:"current"`
	}

	var data respData

	url := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no", conf.weatherApiKey, url.QueryEscape(location))

	err := request(context.Background(), url, &data)
	if err != nil {
		return nil, fmt.Errorf("error requesting weather api: %w", err)
	}

	resp := &WeatherInfo{
		Celsius:    data.Current.TempC,
		Fahrenheit: data.Current.TempF,
		Kelvin:     data.Current.TempC + 273,
	}

	return resp, err
}

func request(ctx context.Context, url string, data any) error {

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("error to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error to do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error to read body: %w", err)
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		return fmt.Errorf("error to unmarshal body: %w", err)
	}

	return nil
}
