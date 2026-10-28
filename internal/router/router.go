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
	goalRepo := repository.NewGoalRepository(db)
	settlementRepo := repository.NewSettlementPaymentRepository(db)

	activityService := service.NewActivityService(activityRepo, billRepo, participantRepo)
	billService := service.NewBillService(billRepo, expenseRepo, participantRepo, paymentRepo, goalRepo, settlementRepo, activityService, db)
	expenseService := service.NewExpenseService(expenseRepo, billRepo, participantRepo, db, activityService)
	participantService := service.NewParticipantService(participantRepo, billRepo, expenseService, activityService, goalRepo, settlementRepo, db)
	paymentService := service.NewPaymentService(paymentRepo, billRepo, participantRepo, db)
	userService := service.NewUserService(userRepo, cfg.JWTSecret)
	goalService := service.NewGoalService(goalRepo, billRepo, participantRepo, activityService, db)
	settlementService := service.NewSettlementService(settlementRepo, billRepo, participantRepo, activityService, db)

	billHandler := handler.NewBillHandler(billService)
	activityHandler := handler.NewActivityHandler(activityService)
	participantHandler := handler.NewParticipantHandler(participantService)
	expenseHandler := handler.NewExpenseHandler(expenseService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	userHandler := handler.NewUserHandler(userService)
	goalHandler := handler.NewGoalHandler(goalService)
	settlementHandler := handler.NewSettlementHandler(settlementService)

	optionalAuth := middleware.OptionalAuthMiddleware(cfg.JWTSecret)
	apiKeyAuth := middleware.ApiKeyAuthMiddleware(cfg.JWTSecret)

	r.GET("/health", healthHandler.Health)

	api := r.Group("/api")
	{
		api.POST("/split-bills", optionalAuth, billHandler.CreateSplitBill)
		api.GET("/split-bills/:public_id", apiKeyAuth, billHandler.GetSplitBill)
		api.POST("/group-bills", optionalAuth, billHandler.CreateGroupBill)
		api.GET("/group-bill/:public_id", apiKeyAuth, billHandler.GetGroupBill)
		api.POST("/split-bills/:public_id/join", billHandler.JoinSplitBill)
		api.PUT("/payment/:public_id", paymentHandler.ProcessPayment)
		api.POST("/settlement/:public_id", middleware.ParticipantMiddleware(), settlementHandler.Create)
		api.DELETE("/settlement/:public_id/:id", middleware.ParticipantMiddleware(), settlementHandler.Delete)
		api.GET("/group-bill/:public_id/activities", activityHandler.GetActivities)
		api.GET("/split-bills/:public_id/activities", activityHandler.GetActivities)

		// Patungan (goal bill): kontribusi bebas, tanpa expense.
		api.POST("/goal-bills", optionalAuth, goalHandler.CreateGoalBill)
		api.GET("/goal-bills/:public_id", goalHandler.GetGoalBill)
		api.POST("/goal-bills/:public_id/join", billHandler.JoinSplitBill)
		api.GET("/goal-bills/:public_id/activities", activityHandler.GetActivities)
		api.PATCH("/goal-bills/:public_id", middleware.ParticipantMiddleware(), goalHandler.UpdateGoalBill)
		api.POST("/goal-bills/:public_id/contributions", middleware.ParticipantMiddleware(), goalHandler.AddContribution)
		api.DELETE("/goal-bills/:public_id/contributions/:id", middleware.ParticipantMiddleware(), goalHandler.DeleteContribution)

		api.POST("/auth/anonymous", userHandler.AuthAnon)
		api.POST("/auth/google", userHandler.AuthGoogle)

		api.GET("/me/bills", apiKeyAuth, billHandler.GetMyBills)

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
