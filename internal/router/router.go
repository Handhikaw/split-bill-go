package router

import (
	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/config"
	"github.com/harkandavez/split-bill-go/internal/handler"
	"github.com/harkandavez/split-bill-go/internal/middleware"
	"github.com/harkandavez/split-bill-go/internal/repository"
	"github.com/harkandavez/split-bill-go/internal/service"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	healthHandler := handler.NewHealthHandler()

	// Dependency Injection manual
	billRepo := repository.NewBillRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	activityRepo := repository.NewActivityRepository(db)
	userRepo := repository.NewUserRepository(db)

	activityService := service.NewActivityService(activityRepo, billRepo, participantRepo)
	billService := service.NewBillService(billRepo, expenseRepo, participantRepo, paymentRepo, activityService, db)
	expenseService := service.NewExpenseService(expenseRepo, billRepo, participantRepo, db, activityService)
	participantService := service.NewParticipantService(participantRepo, billRepo, expenseService, activityService, db)
	paymentService := service.NewPaymentService(paymentRepo, billRepo, participantRepo)
	userService := service.NewUserService(userRepo, cfg.JWTSecret)

	billHandler := handler.NewBillHandler(billService)
	activityHandler := handler.NewActivityHandler(activityService)
	participantHandler := handler.NewParticipantHandler(participantService)
	expenseHandler := handler.NewExpenseHandler(expenseService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	userHandler := handler.NewUserHandler(userService)

	optionalAuth := middleware.OptionalAuthMiddleware(cfg.JWTSecret)

	r.GET("/health", healthHandler.Health)

	api := r.Group("/api")
	{
		api.POST("/split-bills", optionalAuth, billHandler.CreateSplitBill)
		api.GET("/split-bills/:public_id", billHandler.GetSplitBill)
		api.POST("/group-bills", optionalAuth, billHandler.CreateGroupBill)
		api.GET("/group-bill/:public_id", billHandler.GetGroupBill)
		api.POST("/split-bills/:public_id/join", billHandler.JoinSplitBill)
		api.PUT("/payment/:public_id", paymentHandler.ProcessPayment)
		api.GET("/group-bill/:public_id/activities", activityHandler.GetActivities)
		api.GET("/split-bills/:public_id/activities", activityHandler.GetActivities)

		api.POST("/auth/anonymous", userHandler.AuthAnon)
		api.POST("/auth/google", userHandler.AuthGoogle)

		participant := api.Group("/participant")
		participant.Use(middleware.ParticipantMiddleware())
		{
			participant.PUT("/:public_id", participantHandler.Participant)
			participant.DELETE("/:public_id/:id", participantHandler.RemoveParticipant)
		}

		expense := api.Group("/expense")
		expense.Use(middleware.ParticipantMiddleware())
		{
			expense.GET("/category", expenseHandler.GetCategory)
			expense.PUT("/category", expenseHandler.UpsertCategory)
			expense.POST("", expenseHandler.Expense)
			expense.PATCH("/:public_id/:expense_id", expenseHandler.UpdateExpense)
			expense.DELETE("/:public_id/:expense_id", expenseHandler.DeleteExpense)
			expense.POST("/:public_id/:expense_id/item", expenseHandler.ExpenseItem)
			expense.PATCH("/:public_id/:expense_id/item/:item_id", expenseHandler.UpdateExpenseItem)
			expense.DELETE("/:public_id/:expense_id/item/:item_id", expenseHandler.DeleteExpenseItem)
		}
	}

}
