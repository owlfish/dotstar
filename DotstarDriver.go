/*
The dotstar package provides an API for controlling a Dotstar LED strip.

This package has been used to drive a 30 LED Dotstar strip from a Raspberry Pi over the SPI bus.
The API requires an io.Writer that sends the written bytes to the Dotstar strip.  The
github.com/kidoman/embd package provides such an interface as shown in the example.

*/
package dotstar

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
)

// headerSize is the leading header of zeros to start a message
const headerSize = 4

// ledPacketSize is the amount of data used per-LED in the message
const ledPacketSize = 4

// brightnessHeader is the first 3 bits set in a byte at the start of the per-LED brightness information.
const brightnessHeader = 0xE0

// defaultGammaFunc applies the pre-computed 2.8 gamma correction
func defaultGammaFunc(in Colour) (out Colour) {
	out.R = defaultGammaTable[in.R]
	out.G = defaultGammaTable[in.G]
	out.B = defaultGammaTable[in.B]
	out.L = in.L
	return out
}

// defaultGammaTable is a pre-computed table of Gamma 2.8
// Used by defaultGammaFunc
var defaultGammaTable []uint8 = []uint8{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2,
	2, 3, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 5, 5, 5,
	5, 6, 6, 6, 6, 7, 7, 7, 7, 8, 8, 8, 9, 9, 9, 10,
	10, 10, 11, 11, 11, 12, 12, 13, 13, 13, 14, 14, 15, 15, 16, 16,
	17, 17, 18, 18, 19, 19, 20, 20, 21, 21, 22, 22, 23, 24, 24, 25,
	25, 26, 27, 27, 28, 29, 29, 30, 31, 32, 32, 33, 34, 35, 35, 36,
	37, 38, 39, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 50,
	51, 52, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 66, 67, 68,
	69, 70, 72, 73, 74, 75, 77, 78, 79, 81, 82, 83, 85, 86, 87, 89,
	90, 92, 93, 95, 96, 98, 99, 101, 102, 104, 105, 107, 109, 110, 112, 114,
	115, 117, 119, 120, 122, 124, 126, 127, 129, 131, 133, 135, 137, 138, 140, 142,
	144, 146, 148, 150, 152, 154, 156, 158, 160, 162, 164, 167, 169, 171, 173, 175,
	177, 180, 182, 184, 186, 189, 191, 193, 196, 198, 200, 203, 205, 208, 210, 213,
	215, 218, 220, 223, 225, 228, 231, 233, 236, 239, 241, 244, 247, 249, 252, 255}

/*
A Colour represents the colour and luminosity (brightness) for an LED
*/
type Colour struct {
	R, G, B uint8
	// Luminosity (Brightness).  255 is maximum, there are only 32 levels so use increments of 8
	L uint8
}

/*
String returns a description of the colour value.
*/
func (c Colour) String() string {
	return fmt.Sprintf("R: %d G: %d B: %d L: %d (#%02X%02X%02X%02X)", c.R, c.G, c.B, c.L, c.R, c.G, c.B, c.L)
}

/*
Blend returns a new Colour that is Ratio on the way between this and NewColour
*/
func (originalColour Colour) Blend(NewColour Colour, Ratio float32) (result Colour) {
	if Ratio <= 0 {
		return originalColour
	}
	if Ratio >= 1 {
		return NewColour
	}

	result.B = uint8((float32(NewColour.B)-float32(originalColour.B))*Ratio + float32(originalColour.B))
	result.G = uint8((float32(NewColour.G)-float32(originalColour.G))*Ratio + float32(originalColour.G))
	result.R = uint8((float32(NewColour.R)-float32(originalColour.R))*Ratio + float32(originalColour.R))
	result.L = uint8((float32(NewColour.L)-float32(originalColour.L))*Ratio + float32(originalColour.L))
	return result
}

// Off represents an unlit LED
var Off Colour = Colour{R: 0, G: 0, B: 0, L: 0}

// Red is full luminosity red.
var Red Colour = Colour{R: 255, G: 0, B: 0, L: 255}

// Green is full luminosity green.
var Green Colour = Colour{R: 0, G: 255, B: 0, L: 255}

// Blue is full luminosity blue.
var Blue Colour = Colour{R: 0, G: 0, B: 255, L: 255}

// White is full luminosity white.
var White Colour = Colour{R: 255, G: 255, B: 255, L: 255}

/*
NewColour builds a new Colour value from the given components.
*/
func NewColour(red, green, blue, luminosity uint8) Colour {
	return Colour{
		R: red,
		G: green,
		B: blue,
		L: luminosity,
	}
}

/*
NewColourFromHex takes a string in hex #RGBL or #RGB format and returns that colour.
*/
func NewColourFromStr(clr string) Colour {
	newClr := Colour{
		R: 0,
		G: 0,
		B: 0,
		L: 255,
	}

	if len(clr) == 7 {
		fmt.Sscanf(clr, "#%02X%02X%02X", &newClr.R, &newClr.G, &newClr.B)
	}

	if len(clr) == 9 {
		fmt.Sscanf(clr, "#%02X%02X%02X%02X", &newClr.R, &newClr.G, &newClr.B, &newClr.L)
	}
	return newClr
}

// ConfigFunc functions are used to change internal configuration of a Controller on creation.
type ConfigFunc func(ctl *Controller)

// OrderConfig returns a configuration function to set the order of the RGB elements in the LED strip.
// The default order is bgr (Blue, Green then Red)
func OrderConfig(order string) (ConfigFunc, error) {
	// TODO - add validation of the order string
	lowerOrder := strings.ToLower(order)
	rOrder := strings.IndexAny(lowerOrder, "r")
	gOrder := strings.IndexAny(lowerOrder, "g")
	bOrder := strings.IndexAny(lowerOrder, "b")

	if rOrder == -1 || gOrder == -1 || bOrder == -1 {
		return nil, errors.New("Order configuration must contain rgb")
	}

	if len(order) != 3 {
		return nil, errors.New("Additional characters other than rgb are not supported")
	}

	return func(ctl *Controller) {
		// +1 to account for the brightness byte at the start
		ctl.rOffset = rOrder + 1
		ctl.gOffset = gOrder + 1
		ctl.bOffset = bOrder + 1
	}, nil
}

/*
DisableGammaCorrectionConfig is used to disable automatic gamma correction.
The Controller defaults to a gamma correction value of 2.8 when writing out colours to the LEDs.
*/
func DisableGammaCorrectionConfig() ConfigFunc {
	return func(ctl *Controller) {
		ctl.gammaFunc = nil
	}
}

/*
SetCustomGammaCorrectionConfig allows a customer Gamma correction function to be used.

The gammafunc is called each time a value is written to the LEDs with Update()
*/
func SetCustomGammaCorrectionConfig(gammafunc func(in Colour) Colour) ConfigFunc {
	return func(ctl *Controller) {
		ctl.gammaFunc = gammafunc
	}
}

// defaultOrder is default configuration for ordering the colours.
var defaultOrder, _ = OrderConfig("bgr")

// defaultGamma uses a table of pre-computed gamma values using a global 2.8 value
var defaultGamma = SetCustomGammaCorrectionConfig(defaultGammaFunc)

/*
A Controller provides a friendly interface to the Dotstar stip of LEDs.

Methods are NOT safe to call from multiple goroutines concurrently.
*/
type Controller struct {
	// spi is a Writer to the underlying SPI
	spi io.Writer
	// ledColours holds the value that will be written to the LEDs on next Update()
	ledColours []Colour
	// buffer is used to construct the message stream to be sent in Update()
	buffer []byte
	// count is the number of LEDs in this Dotstar string
	count int
	// rOffset, gOffset and bOffset hold the order of Red, Green and Blue to be used when writing the colour data
	rOffset, gOffset, bOffset int
	// brightness holds the global brightness which is used to scale the per-LED Luminosity values.
	brightness uint8
	// gammaFunc may be nil (no gamma applied) or a function that pre-processes the Colour to apply gamma correction.
	// The function is call when preparing the buffer contents.
	gammaFunc func(Colour) Colour
}

/*
NewController creates a new Controller.

SpiOut is the io.Writer that sends written bytes to the Dotstar LED strip.
LedCount is the number of LEDs present in the strip.

The default settings are for LED colours to be  written out bgr and a gamma correction of 2.8 to be applied.
Pass ConfigFunc values to override these settings.
*/
func NewController(SpiOut io.Writer, LedCount int, cfgs ...ConfigFunc) *Controller {
	// footerSize := int(math.Ceil(float64(LedCount-1)/16)) + 2
	footerSize := int(math.Ceil(float64(LedCount-1)/16)) + 2
	ctl := &Controller{
		spi:        SpiOut,
		count:      LedCount,
		ledColours: make([]Colour, LedCount, LedCount),
		buffer:     make([]byte, headerSize+LedCount*ledPacketSize+footerSize, headerSize+LedCount*ledPacketSize+footerSize),
		brightness: 255,
	}

	defaultOrder(ctl)
	defaultGamma(ctl)

	for _, cfg := range cfgs {
		cfg(ctl)
	}

	return ctl
}

/*
Update sends the current Colour values to the LEDs.
*/
func (ctl *Controller) Update() error {
	n, err := ctl.spi.Write(ctl.buffer)
	if err != nil {
		return err
	}
	if n != len(ctl.buffer) {
		return errors.New("Unable to send the full LED colour buffer to device")
	}
	return nil
}

/*
Clear turns off all LEDs.  This does not trigger Update().
*/
func (ctl *Controller) Clear() {
	for i := 0; i < ctl.count; i++ {
		ctl.SetColour(i, Off)
	}
}

/*
Snapshot returns a copy of the currently set colours.

These colours may not have been sent to the LEDs yet with Update().
*/
func (ctl *Controller) Snapshot() []Colour {
	result := make([]Colour, ctl.count, ctl.count)
	copy(result, ctl.ledColours)
	return result
}

/*
SetGlobalBrightness scales the maximum brightness of any colour to be capped at the given value.

Only the top 5 bits are useful, so a minimum increment change is 8.
Set to 255 for maximum brightness.
*/
func (ctl *Controller) SetGlobalBrightness(brightness uint8) {
	ctl.brightness = brightness

	// Update the buffer to reflect this.
	for i, clr := range ctl.ledColours {
		ctl.updateBuffer(i, clr)
	}
}

/*
GetGlobalBrightness returns the maximum brightness that is shown on the strip.

Only the top 5 bits are useful, so a minimum increment change is 8.
255 is maximum brightness.
*/
func (ctl *Controller) GetGlobalBrightness() uint8 {
	return ctl.brightness
}

/*
SetColour records the Colour that an LED should be set to at the next Update().

If position is out of bounds, no update is made.
*/
func (ctl *Controller) SetColour(position int, colour Colour) {
	if position >= ctl.count || position < 0 {
		// Do nothing - out of bounds
		return
	}

	ctl.ledColours[position] = colour

	ctl.updateBuffer(position, colour)
}

/*
SetColours updates all of the LED colours to the values given.

If more Colour values are given than LEDs, the additional Colours are ignored.
*/
func (ctl *Controller) SetColours(clrs []Colour) {
	for pos, c := range clrs {
		if pos >= ctl.count {
			return
		}
		ctl.SetColour(pos, c)
	}
}

/*
GetColour retrieves the previously set colour of an LED in the given position.

This value may not have been written to the LED yet.
If position is out of bounds then a zero value Colour is returned
*/
func (ctl *Controller) GetColour(position int) Colour {
	var result Colour

	if position >= ctl.count || position < 0 {
		// Do nothing - out of bounds
		return result
	}

	return ctl.ledColours[position]
}

/*
Internal method used to update the buffer to reflect the given colour and global brightness.
*/
func (ctl *Controller) updateBuffer(position int, colour Colour) {
	bufferOffset := headerSize + position*ledPacketSize
	// Write out the brightness
	brightness := colour.L
	if ctl.brightness != 255 {
		brightness = uint8(float32(ctl.brightness) * float32(brightness) / 255)
	}
	if ctl.gammaFunc != nil {
		// Apply gamma correction.
		colour = ctl.gammaFunc(colour)
	}
	ctl.buffer[bufferOffset] = brightness>>3 | brightnessHeader
	ctl.buffer[bufferOffset+ctl.rOffset] = colour.R
	ctl.buffer[bufferOffset+ctl.bOffset] = colour.B
	ctl.buffer[bufferOffset+ctl.gOffset] = colour.G
}
