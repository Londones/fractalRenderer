package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/cmplx"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Complex struct {
	Real float64 `json:"real"`
	Imag float64 `json:"imag"`
}

type JuliaParams struct {
	C             Complex `json:"c"`
	Center        Complex `json:"center"`
	Zoom          float64 `json:"zoom"`
	Coloring      int     `json:"coloring"`
	MaxIterations int     `json:"maxIterations"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	LOD           int     `json:"lod"`
}

type Tile struct {
	X, Y         int
	Width        int
	Height       int
	LOD          int
	Image        []byte
	LastAccessed time.Time
}

const (
	TileSize     = 128
	MaxCacheSize = 1000
)

type TileCache struct {
	cache map[string]*Tile
	mutex sync.RWMutex
}

func NewTileCache() *TileCache {
	return &TileCache{
		cache: make(map[string]*Tile),
	}
}

func (tc *TileCache) Get(key string) (*Tile, bool) {
	tc.mutex.RLock()
	tile, exists := tc.cache[key]
	tc.mutex.RUnlock()
	return tile, exists
}

func (tc *TileCache) Set(key string, tile *Tile) {
	tc.mutex.Lock()
	tc.cache[key] = tile
	tc.mutex.Unlock()
}

func (tc *TileCache) Delete(key string) {
	tc.mutex.Lock()
	delete(tc.cache, key)
	tc.mutex.Unlock()
}

func (tc *TileCache) Len() int {
	tc.mutex.RLock()
	length := len(tc.cache)
	tc.mutex.RUnlock()
	return length
}

type TileRequest struct {
	X, Y int
	LOD  int
}

type Offset struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

var tileCache = NewTileCache()

func generateJuliaTile(params JuliaParams, tileX, tileY, tileWidth, tileHeight, lod int, offset Offset) *Tile {
	img := image.NewRGBA(image.Rect(0, 0, tileWidth, tileHeight))
	c := complex(params.C.Real, params.C.Imag)

	for py := 0; py < tileHeight; py++ {
		y := (float64(tileY+py)-offset.Y)/params.Zoom - float64(params.Height)/(2*params.Zoom) + params.Center.Imag
		for px := 0; px < tileWidth; px++ {
			x := (float64(tileX+px)-offset.X)/params.Zoom - float64(params.Width)/(2*params.Zoom) + params.Center.Real
			z := complex(x, y)

			var i int
			for i = 0; i < params.MaxIterations; i++ {
				if cmplx.Abs(z) > 2 {
					break
				}
				z = z*z + c
			}

			if i < params.MaxIterations {
				rgba := ReturnRGBA(params.Coloring, i, params.MaxIterations, z, tileX+px, tileY+py)
				img.Set(px, py, rgba)
			} else {
				img.Set(px, py, color.Black)
			}
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)

	return &Tile{X: tileX, Y: tileY, Width: tileWidth, Height: tileHeight, LOD: lod, Image: buf.Bytes(), LastAccessed: time.Now()}
}

func getTileKey(params JuliaParams, x, y, lod int, offset Offset) string {
	return fmt.Sprintf("%f,%f,%f,%f,%f,%d,%d,%d,%d,%d,%d,%d,%f,%f",
		params.C.Real, params.C.Imag, params.Center.Real, params.Center.Imag,
		params.Zoom, params.MaxIterations, params.Width, params.Height,
		params.Coloring, x, y, lod, offset.X, offset.Y)
}

func getCachedTile(params JuliaParams, x, y, lod int, offset Offset) (*Tile, bool) {
	key := getTileKey(params, x, y, lod, offset)
	tile, exists := tileCache.Get(key)
	if exists {
		tile.LastAccessed = time.Now()
		tileCache.Set(key, tile)
	}
	return tile, exists
}

func cacheTile(params JuliaParams, tile *Tile, offset Offset) {
	key := getTileKey(params, tile.X, tile.Y, tile.LOD, offset)
	tileCache.Set(key, tile)

	if tileCache.Len() > MaxCacheSize {
		var oldestKey string
		var oldestTime time.Time
		tileCache.mutex.RLock()
		for k, v := range tileCache.cache {
			if oldestKey == "" || v.LastAccessed.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.LastAccessed
			}
		}
		tileCache.mutex.RUnlock()
		tileCache.Delete(oldestKey)
	}
}

func generateJuliaSet(params JuliaParams, ch chan<- *Tile, requestedTiles []TileRequest, offset Offset) {
	var wg sync.WaitGroup

	// Sort requested tiles by LOD (ascending) and then by distance from center
	sort.Slice(requestedTiles, func(i, j int) bool {
		if requestedTiles[i].LOD != requestedTiles[j].LOD {
			return requestedTiles[i].LOD < requestedTiles[j].LOD
		}
		centerX, centerY := params.Width/2, params.Height/2
		distI := math.Pow(float64(requestedTiles[i].X-centerX), 2) + math.Pow(float64(requestedTiles[i].Y-centerY), 2)
		distJ := math.Pow(float64(requestedTiles[j].X-centerX), 2) + math.Pow(float64(requestedTiles[j].Y-centerY), 2)
		return distI < distJ
	})

	for _, tile := range requestedTiles {
		wg.Add(1)
		go func(tx, ty, lod int) {
			defer wg.Done()
			tileWidth := int(math.Min(float64(TileSize), float64(params.Width-tx)))
			tileHeight := int(math.Min(float64(TileSize), float64(params.Height-ty)))

			if cachedTile, exists := getCachedTile(params, tx, ty, lod, offset); exists {
				ch <- cachedTile
			} else {
				tile := generateJuliaTile(params, tx, ty, tileWidth, tileHeight, lod, offset)
				cacheTile(params, tile, offset)
				ch <- tile
			}
		}(tile.X, tile.Y, tile.LOD)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			return
		}

		var request struct {
			Params JuliaParams `json:"params"`
			Tiles  []string    `json:"tiles"`
			Offset Offset      `json:"offset"`
		}
		err = json.Unmarshal(p, &request)
		if err != nil {
			log.Printf("Error unmarshaling JSON: %v", err)
			return
		}

		params := request.Params
		requestedTiles := make([]TileRequest, 0, len(request.Tiles))
		for _, tile := range request.Tiles {
			var x, y int
			fmt.Sscanf(tile, "%d,%d", &x, &y)
			requestedTiles = append(requestedTiles, TileRequest{X: x, Y: y, LOD: params.LOD})
		}

		// Validate params
		if params.LOD < 1 {
			params.LOD = 1
		}

		ch := make(chan *Tile)
		go generateJuliaSet(params, ch, requestedTiles, request.Offset)

		for tile := range ch {
			message := map[string]interface{}{
				"type":   "tile",
				"x":      tile.X,
				"y":      tile.Y,
				"width":  tile.Width,
				"height": tile.Height,
				"lod":    tile.LOD,
				"image":  base64.StdEncoding.EncodeToString(tile.Image),
			}
			err = conn.WriteJSON(message)
			if err != nil {
				log.Printf("Error sending tile: %v", err)
				return
			}
		}
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":8080", nil)
}
