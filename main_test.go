package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func Test_getWeatherData(t *testing.T) {
	resp, err := getWeatherData("Belo Horizonte")

	if err != nil {
		t.Errorf("erro to get weather data: %s", err)
	}

	if resp.Kelvin-resp.Celsius != 273 {
		t.Error("Kelvin temperature wrong")
	}

	if resp.Celsius == 0 && resp.Fahrenheit == 0 {
		t.Error("api response error")
	}
}

func Test_getLocationData(t *testing.T) {
	resp, err := getLocationData("30280160")

	if err != nil {
		t.Errorf("erro to get location data: %s", err)
	}

	if resp.City != "Belo Horizonte" {
		t.Error("api response error")
	}
}

func Test_getLocationData_notFound(t *testing.T) {
	resp, err := getLocationData("99280160")

	if err != nil {
		t.Errorf("erro to get location data: %s", err)
	}

	if resp != nil {
		t.Error("api should return empty")
	}
}

func setUrlParamInChiContext(r *http.Request, key, value string) *http.Request {
	chiCtx := chi.NewRouteContext()
	req := r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
	chiCtx.URLParams.Add(key, value)
	return req
}

func Test_weatherHandler_cepInvalid(t *testing.T) {
	cep := "30280-100"

	r := httptest.NewRequest(http.MethodGet, "/weather/"+cep, nil)
	w := httptest.NewRecorder()

	r = setUrlParamInChiContext(r, "cep", cep)

	weatherHandler(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d, wanted %d", w.Code, http.StatusUnprocessableEntity)
	}
}

// Os testes abaixo fazem uma requisição real para as APIs externas,
// por isso pode ocorrer erro inesperado se a API externa estiver com problemas.
// Pelo que entendi da orientação do desafio,
// mesmo com esse risco testes assim devem existir para avaliação

func Test_weatherHandler_cepNotFound(t *testing.T) {
	cep := "90280100"
	r := httptest.NewRequest(http.MethodGet, "/weather/"+cep, nil)
	w := httptest.NewRecorder()

	r = setUrlParamInChiContext(r, "cep", cep)

	weatherHandler(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, wanted %d", w.Code, http.StatusNotFound)
	}
}

func Test_weatherHandler_success(t *testing.T) {
	cep := "30280160"
	r := httptest.NewRequest(http.MethodGet, "/weather/"+cep, nil)
	w := httptest.NewRecorder()

	r = setUrlParamInChiContext(r, "cep", cep)

	weatherHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("got %d, wanted %d", w.Code, http.StatusOK)
	}
}
