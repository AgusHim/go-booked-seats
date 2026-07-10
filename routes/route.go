package routes

import (
	"strconv"
	"sync"
	"time"

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
	settingRepo := repositories.NewSettingRepository(db)
	settingService := services.NewSettingService(settingRepo)
	settingController := controllers.NewSettingController(settingService)
	ticketService := services.NewTicketService(ticketRepo, settingRepo)
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
	admin_api := app.Group("/admin_api", middleware.AdminProtected(db))
	api := app.Group("/api")
	loginLimiter := simpleRateLimiter(5, time.Minute)
	verifyLimiter := simpleRateLimiter(10, time.Minute)
	pdfLimiter := simpleRateLimiter(3, time.Minute)
	lockLimiter := simpleRateLimiter(120, time.Minute)
	confirmLimiter := simpleRateLimiter(60, time.Minute)
	lockTicketLimiter := simpleTicketRateLimiter(30, time.Minute)
	confirmTicketLimiter := simpleTicketRateLimiter(10, time.Minute)

	api.Post("/login", loginLimiter, userController.Login)
	api.Get("/events", eventController.GetEvents)
	api.Get("/events/:id", eventController.GetEvent)

	admin_api.Get("/users", userController.FindAll)
	admin_api.Post("/users", userController.Create)
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
	admin_api.Post("/tickets/goodie-bags/claim", ticketController.MarkGoodieBagsClaimed)
	admin_api.Post("/tickets/:id/goodie-bag", ticketController.ToggleGoodieBag)
	admin_api.Post("/seats/layout", seatController.SaveBulkLayout)
	admin_api.Get("/settings/darisini", settingController.GetDarisini)
	admin_api.Put("/settings/darisini", settingController.UpdateDarisini)

	importController := controllers.NewImportController(db)
	admin_api.Post("/import-excel", importController.UploadExcel)

	// War Kursi: Public verification endpoint (no admin auth required)
	verifyController := controllers.NewVerifyController(ticketService)
	api.Post("/verify-ticket", verifyLimiter, verifyController.VerifyTicket)
	api.Post("/verify-ticket-pdf", pdfLimiter, verifyController.VerifyTicketPDF)                                                // War kursi: verify via PDF
	api.Post("/seats/lock", lockLimiter, middleware.TicketProtected(db), lockTicketLimiter, seatController.LockSeat)            // War kursi: ticket-auth seat locking
	api.Post("/seats/confirm", confirmLimiter, middleware.TicketProtected(db), confirmTicketLimiter, bookedSeat.ConfirmBooking) // War kursi: ticket-auth permanent booking

	booked := app.Group("/api/booked-seats")

	booked.Get("/", bookedSeat.GetPublic)
	booked.Get("/me", middleware.TicketProtected(db), bookedSeat.GetMine)
	admin_api.Get("/booked-seats", bookedSeat.GetAll)
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

type rateLimitBucket struct {
	count   int
	resetAt time.Time
}

func simpleRateLimiter(max int, window time.Duration) fiber.Handler {
	return simpleRateLimiterBy(max, window, func(c *fiber.Ctx) string {
		return c.IP() + ":" + c.Path()
	})
}

func simpleTicketRateLimiter(max int, window time.Duration) fiber.Handler {
	return simpleRateLimiterBy(max, window, func(c *fiber.Ctx) string {
		ticketID, _ := c.Locals("ticket_id").(string)
		return "ticket:" + ticketID + ":" + c.Path()
	})
}

func simpleRateLimiterBy(max int, window time.Duration, keyFn func(*fiber.Ctx) string) fiber.Handler {
	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)

	return func(c *fiber.Ctx) error {
		now := time.Now()
		key := keyFn(c)

		mu.Lock()
		bucket := buckets[key]
		if bucket.resetAt.IsZero() || now.After(bucket.resetAt) {
			bucket = rateLimitBucket{resetAt: now.Add(window)}
		}
		bucket.count++
		buckets[key] = bucket
		remaining := max - bucket.count
		resetAt := bucket.resetAt
		mu.Unlock()

		c.Set("X-RateLimit-Limit", strconv.Itoa(max))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(maxInt(remaining, 0)))
		c.Set("X-RateLimit-Reset", strconv.Itoa(int(time.Until(resetAt).Seconds())))

		if bucket.count > max {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"message": "Too many requests",
			})
		}

		return c.Next()
	}
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
