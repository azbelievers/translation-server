package main

import (
	"abc-dev/rtc-server/rtc"
	"flag"
	"fmt"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	cors "github.com/rs/cors/wrapper/gin"
)

func main() {
	port := flag.Int("port", 8080, "port to serve on")
	flag.Parse()

	publisherChan := make(chan rtc.Payload)
	trackChan := make(chan *webrtc.TrackLocalStaticRTP)

	mediaEngine := webrtc.MediaEngine{}
	if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
		panic(err)
	}

	interceptor := interceptor.Registry{}
	// Use the default set of Interceptors
	if err := webrtc.RegisterDefaultInterceptors(&mediaEngine, &interceptor); err != nil {
		panic(err)
	}

	peerConnectionConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302", "stun:stun.stunprotocol.org"},
			},
		},
	}

	corsConfig := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Set the router as the default one provided by Gin
	router := gin.Default()
	router.Use(corsConfig)
	router.Use(static.Serve("/", static.LocalFile("./client", true)))

	publisherHandler := rtc.NewPublisherHandler(&publisherChan, trackChan, peerConnectionConfig)
	consumerHandler := rtc.NewConsumerHandler(trackChan, peerConnectionConfig)

	go publisherHandler.Listen(&interceptor, &mediaEngine)
	go consumerHandler.ListenForTrack()

	// Routes

	router.POST("/publisher", func(ctx *gin.Context) {

		var body rtc.Payload

		err := ctx.BindJSON(&body)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		publisherChan <- body

		result := <-publisherHandler.ReturnChannel()
		if result.Error != nil {
			ctx.JSON(500, gin.H{"error": result.Error.Error()})
			return
		}

		ctx.JSON(200, gin.H{"offer": result.Offer})

	})

	router.POST("/consumer", func(ctx *gin.Context) {

		var body rtc.Payload

		err := ctx.BindJSON(&body)
		if err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		go consumerHandler.CreateConsumer(body)

		result := <-consumerHandler.ReturnChannel()
		if result.Error != nil {
			ctx.JSON(500, gin.H{"error": result.Error.Error()})
			return
		}

		ctx.JSON(200, gin.H{"offer": result.Offer})
	})

	// Start server
	router.Run(":" + fmt.Sprint(*port))
}
