package main

import (
	"image/color"
	"math"
	"math/cmplx"

	"github.com/lucasb-eyer/go-colorful"
)

func SmoothColorHSV(i, maxIterations int, z complex128) color.RGBA {
	if i == maxIterations {
		return color.RGBA{0, 0, 0, 255}
	}

	smoothColor := float64(i) - math.Log(math.Log(cmplx.Abs(z)))/math.Log(2)
	hue := math.Sin(smoothColor * 0.1)
	//saturation := 0.8 + 0.2*math.Cos(smoothColor*0.1)
	//value := 1.0 - math.Pow(float64(i)/float64(maxIterations), 0.1)

	c := colorful.Hsv(hue*360, 0.8, 1.0)
	r, g, b := c.RGB255()
	return color.RGBA{r, g, b, 255}
}

func StripePattern(i, maxIterations int, z complex128) color.RGBA {
	if i == maxIterations {
		return color.RGBA{0, 0, 0, 255}
	}

	stripeWidth := 10
	stripeIndex := i / stripeWidth
	inStripe := i % stripeWidth

	baseHue := float64(stripeIndex%6) / 6.0
	hue := baseHue + float64(inStripe)/(float64(stripeWidth)*6.0)
	saturation := 0.8 + 0.2*math.Sin(float64(i)*0.1)
	value := 1.0 - 0.5*float64(inStripe)/float64(stripeWidth)

	c := colorful.Hsv(hue*360, saturation, value)
	r, g, b := c.RGB255()
	return color.RGBA{r, g, b, 255}
}

func ElectricPlasma(i, maxIterations int, z complex128) color.RGBA {
	if i == maxIterations {
		return color.RGBA{0, 0, 0, 255}
	}

	smoothColor := float64(i) - math.Log(math.Log(cmplx.Abs(z)))/math.Log(2)
	t := smoothColor / float64(maxIterations)

	r := uint8(math.Sin(t*math.Pi)*127 + 128)
	g := uint8(math.Sin(t*math.Pi*2)*127 + 128)
	b := uint8(math.Sin(t*math.Pi*4)*127 + 128)

	return color.RGBA{r, g, b, 255}
}

func PsychedelicSwirl(i, maxIterations int, z complex128) color.RGBA {
	if i == maxIterations {
		return color.RGBA{0, 0, 0, 255}
	}

	angle := cmplx.Phase(z)
	radius := cmplx.Abs(z)

	hue := (math.Atan2(angle, math.Log(radius)) + math.Pi) / (2 * math.Pi)
	saturation := 0.8 + 0.2*math.Sin(float64(i)*0.1)
	value := 1.0 - math.Pow(float64(i)/float64(maxIterations), 0.3)

	c := colorful.Hsv(hue*360, saturation, value)
	r, g, b := c.RGB255()
	return color.RGBA{r, g, b, 255}
}

func MetallicSheen(i, maxIterations int, z complex128) color.RGBA {
	if i == maxIterations {
		return color.RGBA{0, 0, 0, 255}
	}

	smoothColor := float64(i) - math.Log(math.Log(cmplx.Abs(z)))/math.Log(2)
	t := smoothColor / float64(maxIterations)

	r := uint8(128 + 127*math.Sin(t*2*math.Pi+0))
	g := uint8(128 + 127*math.Sin(t*2*math.Pi+2*math.Pi/3))
	b := uint8(128 + 127*math.Sin(t*2*math.Pi+4*math.Pi/3))

	return color.RGBA{r, g, b, 255}
}

func RainbowSpiral(i, maxIterations int, z complex128) color.RGBA {
	if i == maxIterations {
		return color.RGBA{0, 0, 0, 255}
	}

	smoothColor := float64(i) - math.Log(math.Log(cmplx.Abs(z)))/math.Log(2)
	t := smoothColor / float64(maxIterations)

	baseColor := colorful.Hsv(t*360, 1.0, 1.0)
	r, g, b := baseColor.RGB255()

	intensity := uint8(255 * math.Pow(1.0-t, 3))
	return color.RGBA{r, g, b, intensity}
}

func AutumnLeaves(i, maxIterations int, z complex128) color.RGBA {
	if i == maxIterations {
		return color.RGBA{0, 0, 0, 255}
	}

	smoothColor := float64(i) - math.Log(math.Log(cmplx.Abs(z)))/math.Log(2)
	t := smoothColor / float64(maxIterations)

	hue := 30 + 60*math.Sin(t*math.Pi)
	saturation := 0.8 + 0.2*math.Cos(t*2*math.Pi)
	value := 0.7 + 0.3*math.Sin(t*4*math.Pi)

	c := colorful.Hsv(hue, saturation, value)
	r, g, b := c.RGB255()
	return color.RGBA{r, g, b, 255}
}

func OceanDepths(i, maxIterations int, z complex128) color.RGBA {
	if i == maxIterations {
		return color.RGBA{0, 0, 0, 255}
	}

	smoothColor := float64(i) - math.Log(math.Log(cmplx.Abs(z)))/math.Log(2)
	t := smoothColor / float64(maxIterations)

	hue := 180 + 60*math.Sin(t*math.Pi)
	saturation := 0.7 + 0.3*math.Cos(t*2*math.Pi)
	value := 0.5 + 0.5*math.Pow(t, 0.5)

	c := colorful.Hsv(hue, saturation, value)
	r, g, b := c.RGB255()
	return color.RGBA{r, g, b, 255}
}

func MoltenLava(i, maxIterations int, z complex128) color.RGBA {
	if i == maxIterations {
		return color.RGBA{0, 0, 0, 255}
	}

	smoothColor := float64(i) - math.Log(math.Log(cmplx.Abs(z)))/math.Log(2)
	t := smoothColor / float64(maxIterations)

	r := uint8(255 * math.Pow(t, 0.5))
	g := uint8(128 * math.Pow(t, 2))
	b := uint8(64 * math.Pow(t, 4))

	return color.RGBA{r, g, b, 255}
}

func BlendTwoAlgorithms(i, maxIterations int, z complex128) color.RGBA {
	color1 := SmoothColorHSV(i, maxIterations, z)
	color2 := MetallicSheen(i, maxIterations, z)

	blendFactor := math.Sin(float64(i) / float64(maxIterations) * math.Pi)

	r := uint8(float64(color1.R)*blendFactor + float64(color2.R)*(1-blendFactor))
	g := uint8(float64(color1.G)*blendFactor + float64(color2.G)*(1-blendFactor))
	b := uint8(float64(color1.B)*blendFactor + float64(color2.B)*(1-blendFactor))

	return color.RGBA{r, g, b, 255}
}

func AlternateAlgorithms(i, maxIterations int, z complex128, x, y int) color.RGBA {
	if (x+y)%2 == 0 {
		return ElectricPlasma(i, maxIterations, z)
	} else {
		return AutumnLeaves(i, maxIterations, z)
	}
}

func MixMultipleAlgorithms(i, maxIterations int, z complex128) color.RGBA {
	color1 := StripePattern(i, maxIterations, z)
	color2 := MetallicSheen(i, maxIterations, z)
	color3 := OceanDepths(i, maxIterations, z)

	t := float64(i) / float64(maxIterations)
	weight1 := math.Sin(t * math.Pi)
	weight2 := math.Sin(t * 2 * math.Pi)
	weight3 := math.Sin(t * 4 * math.Pi)
	totalWeight := weight1 + weight2 + weight3

	r := uint8((float64(color1.R)*weight1 + float64(color2.R)*weight2 + float64(color3.R)*weight3) / totalWeight)
	g := uint8((float64(color1.G)*weight1 + float64(color2.G)*weight2 + float64(color3.G)*weight3) / totalWeight)
	b := uint8((float64(color1.B)*weight1 + float64(color2.B)*weight2 + float64(color3.B)*weight3) / totalWeight)

	return color.RGBA{r, g, b, 255}
}

func createGreyPalette() []int {
	palette := make([]int, 256)
	for i := 0; i < 256; i++ {
		palette[i] = int(uint8(i + 512 - 512*int(math.Exp(-float64(i)/50))/3))
		palette[i] = palette[i]<<16 | palette[i]<<8 | palette[i]
	}
	palette[255] = 0
	return palette
}

var greyPalette = createGreyPalette()

func GreyPalette(i, maxIterations int, z complex128) color.RGBA {
	if i == maxIterations {
		return color.RGBA{0, 0, 0, 255}
	}
	colorInt := greyPalette[i%256]
	return color.RGBA{
		uint8(colorInt & 0xFF),
		uint8((colorInt >> 8) & 0xFF),
		uint8((colorInt >> 16) & 0xFF),
		255,
	}
}

func ReturnRGBA(coloring int, i int, maxIterations int, z complex128, x int, y int) color.RGBA {
	switch coloring {
	case 1:
		return SmoothColorHSV(i, maxIterations, z)
	case 2:
		return StripePattern(i, maxIterations, z)
	case 3:
		return ElectricPlasma(i, maxIterations, z)
	case 4:
		return PsychedelicSwirl(i, maxIterations, z)
	case 5:
		return MetallicSheen(i, maxIterations, z)
	case 6:
		return RainbowSpiral(i, maxIterations, z)
	case 7:
		return AutumnLeaves(i, maxIterations, z)
	case 8:
		return OceanDepths(i, maxIterations, z)
	case 9:
		return MoltenLava(i, maxIterations, z)
	case 10:
		return GreyPalette(i, maxIterations, z)
	case 11:
		return AlternateAlgorithms(i, maxIterations, z, x, y)
	case 12:
		return MixMultipleAlgorithms(i, maxIterations, z)
	default:
		return SmoothColorHSV(i, maxIterations, z)
	}
}
