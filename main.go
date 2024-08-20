package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/cmplx"
	"net/http"
	"runtime"
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

type Offset struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func calculateJuliaSet(params JuliaParams) []byte {
	data := make([]byte, params.Width*params.Height*4)
	c := complex(params.C.Real, params.C.Imag)

	var wg sync.WaitGroup
	numGoroutines := runtime.NumCPU()
	rowsPerGoroutine := params.Height / numGoroutines

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(startY, endY int) {
			defer wg.Done()
			for y := startY; y < endY; y++ {
				imag := float64(y)/params.Zoom - float64(params.Height)/(2*params.Zoom) + params.Center.Imag
				for x := 0; x < params.Width; x++ {
					real := float64(x)/params.Zoom - float64(params.Width)/(2*params.Zoom) + params.Center.Real
					z := complex(real, imag)

					var i int
					for i = 0; i < params.MaxIterations; i++ {
						if cmplx.Abs(z) > 2 {
							break
						}
						z = z*z + c
					}

					index := (y*params.Width + x) * 4
					color := ReturnRGBA(params.Coloring, i, params.MaxIterations, z, x, y)
					data[index] = color.R
					data[index+1] = color.G
					data[index+2] = color.B
					data[index+3] = color.A
				}
			}
		}(i*rowsPerGoroutine, int(math.Min(float64((i+1)*rowsPerGoroutine), float64(params.Height))))
	}

	wg.Wait()
	return data
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
		}
		err = json.Unmarshal(p, &request)
		if err != nil {
			log.Printf("Error unmarshaling JSON: %v", err)
			return
		}

		params := request.Params

		juliaData := calculateJuliaSet(params)

		err = conn.WriteMessage(websocket.BinaryMessage, juliaData)
		if err != nil {
			log.Printf("Error sending julia data: %v", err)
			return
		}
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":8080", nil)
}
