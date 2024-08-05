package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	//"log"
	"math/cmplx"
	"net/http"

	//"os"
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
	OffsetX       int     `json:"offsetX"`
	OffsetY       int     `json:"offsetY"`
}

type TileMessage struct {
	X    int     `json:"x"`
	Y    int     `json:"y"`
	Zoom float64 `json:"zoom"`
	Data string  `json:"data"` // base64 encoded PNG
}

type Tile struct {
	X, Y     int
	Zoom     float64
	Image    image.Image
	LastUsed time.Time
}

const (
	TileSize     = 256
	MaxCacheSize = 1000
)

var tileCache = sync.Map{}
var tileCacheMutex sync.Mutex

func generateJuliaTile(params JuliaParams, tileX, tileY int) *Tile {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))
	c := complex(params.C.Real, params.C.Imag)

	for py := 0; py < TileSize; py++ {
		y := float64(tileY*TileSize+py)/params.Zoom - float64(params.Height)/(2*params.Zoom) + params.Center.Imag + float64(params.OffsetY)/params.Zoom
		for px := 0; px < TileSize; px++ {
			x := float64(tileX*TileSize+px)/params.Zoom - float64(params.Width)/(2*params.Zoom) + params.Center.Real + float64(params.OffsetX)/params.Zoom
			z := complex(x, y)

			var i int
			for i = 0; i < params.MaxIterations; i++ {
				if cmplx.Abs(z) > 2 {
					break
				}
				z = z*z + c
			}

			if i < params.MaxIterations {

				rgba := ReturnRGBA(params.Coloring, i, params.MaxIterations, z, px, py)

				img.Set(px, py, rgba)
			} else {
				img.Set(px, py, color.Black)
			}
		}
	}
	return &Tile{X: tileX, Y: tileY, Zoom: params.Zoom, Image: img}
}

func getTileKey(params JuliaParams, x, y int) string {
	return fmt.Sprintf("%f,%f,%f,%f,%f,%d,%d,%d,%d,%d,%d,%f,%f",
		params.C.Real, params.C.Imag, params.Center.Real, params.Center.Imag,
		params.Zoom, params.MaxIterations, params.Width, params.Height,
		params.OffsetX, params.OffsetY, params.Coloring, float64(x), float64(y))
}

func cleanupCache(params JuliaParams) {
	tileCacheMutex.Lock()
	defer tileCacheMutex.Unlock()

	var tiles []*Tile
	tileCache.Range(func(_, value interface{}) bool {
		tiles = append(tiles, value.(*Tile))
		return true
	})

	if len(tiles) > MaxCacheSize {
		// Sort tiles by last used time
		sort.Slice(tiles, func(i, j int) bool {
			return tiles[i].LastUsed.Before(tiles[j].LastUsed)
		})

		// Remove oldest tiles
		for i := 0; i < len(tiles)-MaxCacheSize; i++ {
			tileCache.Delete(getTileKey(params, tiles[i].X, tiles[i].Y))
		}
	}
}

func generateJuliaSet(params JuliaParams, modifiedTiles chan<- *Tile) {
	tilesX := (params.Width + TileSize - 1) / TileSize
	tilesY := (params.Height + TileSize - 1) / TileSize

	var wg sync.WaitGroup
	for tileY := 0; tileY < tilesY; tileY++ {
		for tileX := 0; tileX < tilesX; tileX++ {
			wg.Add(1)
			go func(tx, ty int) {
				defer wg.Done()
				key := getTileKey(params, tx, ty)
				if cachedTile, ok := tileCache.Load(key); ok {
					tile := cachedTile.(*Tile)
					tile.LastUsed = time.Now()
					tileCache.Store(key, tile)
					modifiedTiles <- tile
				} else {
					tile := generateJuliaTile(params, tx, ty)
					tile.LastUsed = time.Now()
					tileCache.Store(key, tile)
					modifiedTiles <- tile
				}
			}(tileX, tileY)
		}
	}
	wg.Wait()
	close(modifiedTiles)

	go cleanupCache(params)
}

func encodeTile(tile *Tile) (TileMessage, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, tile.Image)
	if err != nil {
		return TileMessage{}, err
	}
	return TileMessage{
		X:    tile.X,
		Y:    tile.Y,
		Zoom: tile.Zoom,
		Data: base64.StdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
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
			return
		}

		var params JuliaParams
		err = json.Unmarshal(p, &params)
		if err != nil {
			return
		}

		modifiedTiles := make(chan *Tile, 100)
		go generateJuliaSet(params, modifiedTiles)

		for tile := range modifiedTiles {
			tileMsg, err := encodeTile(tile)
			if err != nil {
				continue
			}
			err = conn.WriteJSON(tileMsg)
			if err != nil {
				return
			}
		}
	}
}

func juliaHandler(w http.ResponseWriter, r *http.Request) {
	var params JuliaParams
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	modifiedTiles := make(chan *Tile, 100)
	go generateJuliaSet(params, modifiedTiles)

	fullImage := image.NewRGBA(image.Rect(0, 0, params.Width, params.Height))
	for tile := range modifiedTiles {
		draw.Draw(fullImage, image.Rect(tile.X*TileSize, tile.Y*TileSize, (tile.X+1)*TileSize, (tile.Y+1)*TileSize),
			tile.Image, image.Point{}, draw.Over)
	}

	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, fullImage)

	// // Save the image locally
	// currentTime := time.Now()
	// dateTimeStr := currentTime.Format("020120061504")
	// filename := "julia" + dateTimeStr + ".png"
	// outFile, err := os.Create(filename)
	// if err != nil {
	// 	log.Println("Error creating file:", err)
	// 	return
	// }
	// defer outFile.Close()
	// err = png.Encode(outFile, fullImage)
	// if err != nil {
	// 	log.Println("Error encoding PNG:", err)
	// }
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/julia", juliaHandler)
	http.ListenAndServe(":8080", nil)
}
