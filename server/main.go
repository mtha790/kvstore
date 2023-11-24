package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

type Item struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

type KVStore map[string]Item

func (items KVStore) GetAll() []Item {
	mu.Lock()
	itemList := []Item{}
	for _, item := range items {
		itemList = append(itemList, item)
	}
	mu.Unlock()
	return itemList
}

func (items KVStore) Create(newItem Item) {
	mu.Lock()
	items[newItem.Id] = newItem
	mu.Unlock()
}

func (items KVStore) Get(id string) (Item, bool) {
	mu.Lock()
	item, ok := items[id]
	mu.Unlock()
	return item, ok
}

func (items KVStore) Put(id string, value string) {
	mu.Lock()
	storedItem := items[id]
	storedItem.Value = value
	items[id] = storedItem
	mu.Unlock()
}

func (items KVStore) Delete(id string) {
	mu.Lock()
	delete(items, id)
	mu.Unlock()
}

var (
	STORE = KVStore{}
	mu    sync.Mutex // guards items
)

// Handler for "/items" path
type ItemsHandler struct{}

func (h ItemsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	itemList := STORE.GetAll()
	json.NewEncoder(w).Encode(itemList)
	w.WriteHeader(http.StatusOK)
}

func (h ItemsHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	var newItem Item
	if err := json.NewDecoder(r.Body).Decode(&newItem); err != nil {
		http.Error(w, "Error unmarshaling JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	STORE.Create(newItem)
	w.WriteHeader(http.StatusCreated)
}

func (h ItemsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.handleGet(w, r)
	case "POST":
		h.handlePost(w, r)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

// Http Handler for /item/{id} path
type ItemHandler struct{}

func (h ItemHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/item/"):]
	item, ok := STORE.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	json.NewEncoder(w).Encode(item)
	w.WriteHeader(http.StatusOK)
}

func (h ItemHandler) handlePut(w http.ResponseWriter, r *http.Request) {
	var updItem Item
	if err := json.NewDecoder(r.Body).Decode(&updItem); err != nil {
		http.Error(w, "Error in unmarshaling JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	id := r.URL.Path[len("/item/"):]
	STORE.Put(id, updItem.Value)
	w.WriteHeader(http.StatusOK)
}
func (h ItemHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/item/"):]
	STORE.Delete(id)
	w.WriteHeader(http.StatusOK)
}

func (h ItemHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.handleGet(w, r)
	case "PUT":
		h.handlePut(w, r)
	case "DELETE":
		h.handleDelete(w, r)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

// Entry point
func main() {
	address := flag.String("address", "127.0.0.1", "Server address")
	port := flag.String("port", "8080", "Server port")

	slog.Debug("Register Handlers")
	mux := http.NewServeMux()
	mux.Handle("/items", ItemsHandler{})
	mux.Handle("/item/", ItemHandler{})

	serverAddress := fmt.Sprintf("%s:%s", *address, *port)
	slog.Info("Starting the server", "address", serverAddress)

	err := http.ListenAndServe(serverAddress, mux)
	slog.Error(err.Error())
}
