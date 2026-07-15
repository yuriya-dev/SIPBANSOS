package server

import (
	"github.com/gin-gonic/gin"
	"github.com/wahyutricahya/SIPBANSOS/backend/internal/handler"
	"github.com/wahyutricahya/SIPBANSOS/backend/pkg/middleware"
)

func RegisterRoutes(r *gin.Engine, h *handler.Handler, authMiddleware gin.HandlerFunc) {
	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
		api.GET("/debug-db", h.DebugDB)

		auth := api.Group("/auth")
		{
			auth.POST("/login", h.Login)
			auth.POST("/refresh", h.Refresh)
			auth.GET("/me", authMiddleware, h.Me)
		}

		protected := api.Group("")
		protected.Use(authMiddleware)
		{
			protected.GET("/kriteria/versions", h.ListKriteriaVersions)

			kriteriaGroup := protected.Group("/kriteria", middleware.RequireRoles("admin"))
			{
				kriteriaGroup.GET("", h.ListKriteria)
				kriteriaGroup.GET("/:id", h.GetKriteriaByID)
				kriteriaGroup.POST("", h.CreateKriteria)
				kriteriaGroup.PUT("/:id", h.UpdateKriteria)
				kriteriaGroup.POST("/definitions", h.CreateKriteriaDefinition)
				kriteriaGroup.PUT("/definitions/:id", h.UpdateKriteriaDefinition)
				kriteriaGroup.DELETE("/definitions/:id", h.DeleteKriteriaDefinition)
			}

			importGroup := protected.Group("/import", middleware.RequireRoles("admin", "petugas"))
			{
				importGroup.GET("/template", h.DownloadImportTemplate)
				importGroup.POST("/validate", h.ValidateImport)
				importGroup.POST("/confirm", h.ConfirmImport)
			}

			exportGroup := protected.Group("/export", middleware.RequireRoles("admin", "petugas", "kepala_desa"))
			{
				exportGroup.GET("/data", h.ExportData)
			}

			wargaRead := protected.Group("/warga")
			{
				wargaRead.GET("", h.ListWarga)
				wargaRead.GET("/:id", h.GetWarga)
				wargaRead.GET("/:id/history", h.GetWargaHistory)
			}

			wargaWrite := protected.Group("/warga", middleware.RequireRoles("admin", "petugas"))
			{
				wargaWrite.POST("", h.CreateWarga)
				wargaWrite.PUT("/:id", h.UpdateWarga)
				wargaWrite.DELETE("/:id", h.DeleteWarga)
			}

			protected.POST("/upload", h.UploadFile)

			protected.GET("/reports/weekly-activity", h.GetWeeklyActivity)
			protected.GET("/reports/field-progress", h.GetFieldProgress)
			protected.GET("/reports/periods", h.ListPeriode)
			protected.GET("/reports/summary", h.ReportSummary)

			reportGroup := protected.Group("/reports", middleware.RequireRoles("admin", "kepala_desa"))
			{
				reportGroup.GET("/ranking", h.ReportRanking)
				reportGroup.GET("/rekap", h.ReportRekap)
				reportGroup.GET("/audit", h.ListAuditLogs)
				reportGroup.GET("/export", h.ExportReport)
			}

			protected.POST("/saw/run", middleware.RequireRoles("admin", "kepala_desa"), h.RunSAW)

			settingsGroup := protected.Group("/settings")
			{
				settingsGroup.GET("", h.GetSettings)
				settingsGroup.PUT("", middleware.RequireRoles("admin"), h.UpdateSettings)
			}

			periodsGroup := protected.Group("/periods", middleware.RequireRoles("admin"))
			{
				periodsGroup.POST("", h.CreatePeriode)
				periodsGroup.PUT("/:id", h.UpdatePeriode)
				periodsGroup.DELETE("/:id", h.DeletePeriode)
			}

			schedulesGroup := protected.Group("/schedules")
			{
				schedulesGroup.GET("", h.ListSchedules)
				schedulesGroup.POST("", middleware.RequireRoles("admin"), h.CreateSchedule)
				schedulesGroup.PUT("/:id", middleware.RequireRoles("admin"), h.UpdateSchedule)
				schedulesGroup.DELETE("/:id", middleware.RequireRoles("admin"), h.DeleteSchedule)
			}


			usersGroup := protected.Group("/users", middleware.RequireRoles("admin"))
			{
				usersGroup.GET("", h.ListUsers)
				usersGroup.POST("", h.CreateUser)
				usersGroup.PUT("/:id", h.UpdateUser)
				usersGroup.DELETE("/:id", h.DeleteUser)
			}
		}
	}
}
