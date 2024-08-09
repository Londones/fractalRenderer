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
	"math/cmplx"
	"net/http"
	"sync"

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
}

func generateJuliaSet(params JuliaParams) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, params.Width, params.Height))
	c := complex(params.C.Real, params.C.Imag)

	var wg sync.WaitGroup
	for py := 0; py < params.Height; py++ {
		wg.Add(1)
		go func(y int) {
			defer wg.Done()
			for px := 0; px < params.Width; px++ {
				x := float64(px)/params.Zoom - float64(params.Width)/(2*params.Zoom) + params.Center.Real
				y := float64(y)/params.Zoom - float64(params.Height)/(2*params.Zoom) + params.Center.Imag
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
		}(py)
	}
	wg.Wait()

	return img
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

		var params JuliaParams
		err = json.Unmarshal(p, &params)
		if err != nil {
			log.Printf("Error unmarshaling JSON: %v", err)
			return
		}

		img := generateJuliaSet(params)

		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		if err != nil {
			log.Printf("Error encoding PNG: %v", err)
			return
		}

		base64Image := base64.StdEncoding.EncodeToString(buf.Bytes())
		err = conn.WriteJSON(map[string]string{"image": base64Image})
		if err != nil {
			log.Printf("Error sending image: %v", err)
			return
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

	img := generateJuliaSet(params)

	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, img)
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/julia", juliaHandler)
	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":8080", nil)
}
