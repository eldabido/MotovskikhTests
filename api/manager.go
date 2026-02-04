package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/LarsFox/motovskikh-hse-backend/manager"
)

const (
	defaultReadTimeout  = time.Second * 15
	defaultWriteTimeout = time.Second * 30
	defaultIdleTimeout  = time.Second * 30
)

type Manager struct {
	manager *manager.Manager
	router  *mux.Router
}

type route struct {
	Method   string
	Path     string
	Handler  http.HandlerFunc
	Wrappers []wrapper
}


func routeGet(path string, handler http.HandlerFunc, wrappers ...wrapper) route {
	return route{http.MethodGet, path, handler, wrappers}
}

func routePost(path string, handler http.HandlerFunc, wrappers ...wrapper) route {
	return route{http.MethodPost, path, handler, wrappers}
}

func NewManager(manager *manager.Manager) *Manager {
	m := &Manager{
		manager: manager,
		router:  mux.NewRouter().StrictSlash(true),
	}

	m.addRoutes()
	
	// Swagger
	swaggerHandler := httpSwagger.Handler(
		httpSwagger.URL("/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)
	
	m.router.PathPrefix("/swagger/").Handler(swaggerHandler)
	return m
}

func (m *Manager) Listen(addr string) error {
	log.Println("API started on addr", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      m.router,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	return server.ListenAndServe()
}

func (m *Manager) addRoutes() {
	m.addHandlers([]route{
		routeGet("/doc.json", m.hndlrSwaggerJSON),
		routeGet("/tests/get/", m.hndlrGetTest),           // Получить тест.
		routePost("/tests/submit/", m.hndlrSubmitTest, m.wrapContentTypeJSON), // Отправить ответы.
		routePost("/stats/analysis/", m.hndlrGetAnalysis), // Базовый анализ.
		routePost("/stats/advanced-analysis/", m.hndlrGetAdvancedAnalysis, m.wrapContentTypeJSON), // Подробный анализ.
		// Для разработки.
		routePost("/dev/create-test-data/", m.hndlrCreateTestData), // Создать тестовые данные.
	})
	log.Println("Routes registered")
}

func (m *Manager) addHandlers(routes []route) {
	essentialWrappers := []wrapper{m.wrapBodyMaxSize, m.wrapEasterEggHeader, wrapRecover}
	for _, r := range routes {
		var wrapper http.Handler = r.Handler
		for _, w := range r.Wrappers {
			wrapper = w(wrapper)
		}
		for _, w := range essentialWrappers {
			wrapper = w(wrapper)
		}
		m.router.Methods(r.Method).Path(r.Path).Handler(wrapper)
	}
}

func (m *Manager) send(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	resp := map[string]interface{}{
		"ok":     true,
		"result": data,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		notify(err)
	}
}

func (m *Manager) hndlrSwaggerJSON(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./doc.json")
}
