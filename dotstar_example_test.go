package dotstar_test

import (
	"github.com/kidoman/embd"
	"github.com/owlfish/dotstar"
	"time"
)

func ExampleRedBlue() {
	ledCount := 30

	if err := embd.InitSPI(); err != nil {
		panic(err)
	}
	defer embd.CloseSPI()

	spiSpeed := 31200000
	spiBus := embd.NewSPIBus(embd.SPIMode0, 0, spiSpeed, 8, 0)
	defer spiBus.Close()

	strip := dotstar.NewController(spiBus, ledCount)

	// Set each LED to red and blue alternatively.
	var clr dotstar.Colour
	for i := 0; i < ledCount; i++ {
		if i%2 == 0 {
			clr = dotstar.Blue
		} else {
			clr = dotstar.Red
		}
		strip.SetColour(i, clr)
	}
	strip.Update()
	time.Sleep(time.Hour * 1)
}
