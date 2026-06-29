package events

import (
	"database/sql"
	"net/http"

	"kvm_v2/Events/academic"
	"kvm_v2/Events/attendance"
	authhandler "kvm_v2/Events/auth"
	"kvm_v2/Events/batches"
	"kvm_v2/Events/enrollments"
	"kvm_v2/Events/health"
	"kvm_v2/Events/reports"
	"kvm_v2/Events/sessions"
	"kvm_v2/Events/users"
	"kvm_v2/internal/repository"
	"kvm_v2/services/auth"
	"kvm_v2/services/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Dependencies struct {
	DB   *sql.DB
	Auth *auth.Service
	Cfg  config.Config
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	health.NewHandler(deps.DB).RegisterRoutes(r)

	fileServer := http.FileServer(http.Dir("test_ui"))
	r.Handle("/test_ui/*", http.StripPrefix("/test_ui", fileServer))
	r.Get("/test_ui", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/test_ui/", http.StatusMovedPermanently)
	})

	usersRepo := repository.NewUserRepository(deps.DB)
	academicRepo := repository.NewAcademicRepository(deps.DB)
	batchRepo := repository.NewBatchRepository(deps.DB)
	enrollmentRepo := repository.NewEnrollmentRepository(deps.DB)
	sessionRepo := repository.NewSessionRepository(deps.DB)
	attendanceRepo := repository.NewAttendanceRepository(deps.DB)
	reportRepo := repository.NewReportRepository(deps.DB)

	r.Route("/api/v1", func(r chi.Router) {
		authhandler.NewHandler(usersRepo, deps.Auth).RegisterRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(deps.Auth.Middleware)

			users.NewHandler(usersRepo, deps.Auth).RegisterRoutes(r)
			academic.NewHandler(academicRepo).RegisterRoutes(r)
			batches.NewHandler(batchRepo).RegisterRoutes(r)
			enrollments.NewHandler(enrollmentRepo).RegisterRoutes(r)
			sessions.NewHandler(sessionRepo, batchRepo).RegisterRoutes(r)
			attendance.NewHandler(attendanceRepo, sessionRepo, batchRepo).RegisterRoutes(r)
			reports.NewHandler(reportRepo, batchRepo).RegisterRoutes(r)
		})
	})

	return r
}
