package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"go-ticketing/controllers"
	middleware "go-ticketing/midleware"
	"go-ticketing/repositories"
	"go-ticketing/services"
)

func RegisterRoutes(app *fiber.App, db *gorm.DB, rdb *redis.Client) {

	ws := controllers.NewWebsocketController()

	userRepo := repositories.NewUserRepository(db)
	userService := services.NewUserService(userRepo)
	userController := controllers.NewUserController(userService)

	bookedSeatRepo := repositories.NewBookedSeatRepository(db, rdb)
	bookedSeatService := services.NewBookedSeatService(bookedSeatRepo)
	bookedSeat := controllers.NewBookedSeatController(bookedSeatService, ws)

	seatRepo := repositories.NewSeatRepository(db, rdb)
	seatService := services.NewSeatService(seatRepo)
	seatController := controllers.NewSeatController(seatService, ws)

	ticketRepo := repositories.NewTicketRepository(db)
	ticketService := services.NewTicketService(ticketRepo)
	ticketController := controllers.NewTicketController(ticketService)

	dashboardRepo := repositories.NewDashboardRepository(db)
	dashboardService := services.NewDashboardService(dashboardRepo)
	dashboardController := controllers.NewDashboardController(dashboardService)

	eventRepo := repositories.NewEventRepository(db)
	eventService := services.NewEventService(eventRepo)
	eventController := controllers.NewEventController(eventService)

	// Middleware: WebSocket Upgrade
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// WebSocket Endpoint
	app.Get("/ws", ws.UpgradeConnection)
	admin_api := app.Group("/admin_api", middleware.AuthProtected())
	api := app.Group("/api")
	api.Post("/login", userController.Login)
	api.Post("/users", userController.Create)
	api.Get("/events", eventController.GetEvents)
	api.Get("/events/:id", eventController.GetEvent)

	admin_api.Get("/users", userController.FindAll)
	admin_api.Delete("/users/:id", userController.Delete)
	admin_api.Post("/events", eventController.CreateEvent)
	admin_api.Put("/events/:id", eventController.UpdateEvent)
	admin_api.Delete("/events/:id", eventController.DeleteEvent)

	seat := app.Group("/api/seats")
	seat.Get("/", seatController.GetAll)
	admin_api.Post("/seats/locked", seatController.LockSeat)
	api.Get("/seats/locked", seatController.GetLockedSeats)
	admin_api.Get("/seats/locked", seatController.GetLockedSeats)
	admin_api.Get("/seats/:id", seatController.GetByID)
	admin_api.Post("/seats", seatController.Create)
	admin_api.Put("/seats/:id", seatController.Update)
	admin_api.Delete("/seats/:id", seatController.Delete)
	admin_api.Get("/tickets", ticketController.GetAll)
	admin_api.Get("/tickets/:id", ticketController.GetByID)
	admin_api.Post("/tickets", ticketController.Create)
	admin_api.Put("/tickets/:id", ticketController.Update)
	admin_api.Delete("/tickets/:id", ticketController.Delete)
	admin_api.Post("/tickets/:id/goodie-bag", ticketController.ToggleGoodieBag)
	admin_api.Post("/seats/layout", seatController.SaveBulkLayout)

	importController := controllers.NewImportController(db)
	admin_api.Post("/import-excel", importController.UploadExcel)

	// War Kursi: Public verification endpoint (no admin auth required)
	verifyController := controllers.NewVerifyController(ticketService)
	api.Post("/verify-ticket", verifyController.VerifyTicket)
	api.Post("/verify-ticket-pdf", verifyController.VerifyTicketPDF) // War kursi: verify via PDF
	api.Post("/seats/lock", seatController.LockSeat)       // War kursi: public seat locking
	api.Post("/seats/confirm", bookedSeat.ConfirmBooking) // War kursi: public seat permanent booking

	booked := app.Group("/api/booked-seats")

	booked.Get("/", bookedSeat.GetAll)
	admin_api.Get("/booked-seats/:id", bookedSeat.GetByID)
	admin_api.Post("/booked-seats", bookedSeat.Create)
	admin_api.Put("/booked-seats/:id", bookedSeat.Update)
	admin_api.Delete("/booked-seats/:id", bookedSeat.Delete)
	admin_api.Post("/booked-seats/upsert", bookedSeat.UpsertBookedSeats)

	tickets := admin_api.Group("/tickets")
	tickets.Post("/", ticketController.Create)
	tickets.Get("/", ticketController.GetAll)
	tickets.Get("/:id", ticketController.GetByID)
	tickets.Put("/:id", ticketController.Update)
	tickets.Delete("/:id", ticketController.Delete)
	tickets.Post("/import", ticketController.ImportCSV)

	admin_api.Get("/dashboard", dashboardController.GetDashboardData)
}
