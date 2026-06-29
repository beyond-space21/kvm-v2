package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestEnrollmentAttendanceTransfer(t *testing.T) {
	srv := testDB(t)
	token := login(t, srv, "test-admin@lms.local", "testpass123")

	classesResp := authRequest(t, http.MethodGet, srv.URL+"/api/v1/classes", token, nil)
	defer classesResp.Body.Close()
	var classes []struct {
		ID    string `json:"id"`
		Grade int    `json:"grade"`
	}
	json.NewDecoder(classesResp.Body).Decode(&classes)

	var class11ID string
	for _, c := range classes {
		if c.Grade == 11 {
			class11ID = c.ID
			break
		}
	}
	if class11ID == "" {
		t.Fatal("class 11 not found in seed data")
	}

	subjectsResp := authRequest(t, http.MethodGet, srv.URL+"/api/v1/subjects", token, nil)
	defer subjectsResp.Body.Close()
	var subjects []struct {
		ID   string `json:"id"`
		Code string `json:"code"`
	}
	json.NewDecoder(subjectsResp.Body).Decode(&subjects)

	var physicsID string
	for _, s := range subjects {
		if s.Code == "PHY" {
			physicsID = s.ID
			break
		}
	}

	yearResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/academic-years", token, map[string]any{
		"name": fmt.Sprintf("flow-year-%d", time.Now().UnixNano()), "start_date": "2026-04-01", "end_date": "2027-03-31", "is_active": true,
	})
	defer yearResp.Body.Close()
	var year struct{ ID string `json:"id"` }
	json.NewDecoder(yearResp.Body).Decode(&year)

	offeringResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/offerings", token, map[string]any{
		"academic_year_id": year.ID, "class_id": class11ID, "subject_id": physicsID, "fee_amount": 5000,
	})
	defer offeringResp.Body.Close()
	var offering struct{ ID string `json:"id"` }
	json.NewDecoder(offeringResp.Body).Decode(&offering)

	staffResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/staff", token, map[string]string{
		"email": fmt.Sprintf("flow-teacher-%d@lms.local", time.Now().UnixNano()), "password": "teach123", "name": "Flow Teacher",
	})
	defer staffResp.Body.Close()
	var staff struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	json.NewDecoder(staffResp.Body).Decode(&staff)

	studentResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/students", token, map[string]string{
		"email": fmt.Sprintf("flow-student-%d@lms.local", time.Now().UnixNano()), "password": "stud123", "name": "Flow Student",
	})
	defer studentResp.Body.Close()
	var student struct{ ID string `json:"id"` }
	json.NewDecoder(studentResp.Body).Decode(&student)

	batchAResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/batches", token, map[string]any{
		"offering_id": offering.ID, "name": "Batch A", "teacher_id": staff.ID,
	})
	defer batchAResp.Body.Close()
	var batchA struct{ ID string `json:"id"` }
	json.NewDecoder(batchAResp.Body).Decode(&batchA)

	batchBResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/batches", token, map[string]any{
		"offering_id": offering.ID, "name": "Batch B", "teacher_id": staff.ID,
	})
	defer batchBResp.Body.Close()
	var batchB struct{ ID string `json:"id"` }
	json.NewDecoder(batchBResp.Body).Decode(&batchB)

	enrollResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/enrollments", token, map[string]string{
		"student_id": student.ID, "academic_year_id": year.ID, "offering_id": offering.ID, "batch_id": batchA.ID,
	})
	defer enrollResp.Body.Close()
	if enrollResp.StatusCode != http.StatusCreated {
		t.Fatalf("enroll status: %d", enrollResp.StatusCode)
	}
	var enrollment struct{ ID string `json:"id"` }
	json.NewDecoder(enrollResp.Body).Decode(&enrollment)

	tmplResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/session-templates", token, map[string]any{
		"batch_id": batchA.ID, "teacher_id": staff.ID, "day_of_week": 1, "start_time": "17:00", "end_time": "18:00",
	})
	defer tmplResp.Body.Close()
	if tmplResp.StatusCode != http.StatusCreated {
		t.Fatalf("template status: %d", tmplResp.StatusCode)
	}

	genResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/sessions/generate", token, map[string]string{
		"start_date": "2026-06-01", "end_date": "2026-06-30",
	})
	defer genResp.Body.Close()
	if genResp.StatusCode != http.StatusOK {
		t.Fatalf("generate status: %d", genResp.StatusCode)
	}

	teacherToken := login(t, srv, staff.Email, "teach123")
	todayResp := authRequest(t, http.MethodGet, srv.URL+"/api/v1/sessions/today", teacherToken, nil)
	defer todayResp.Body.Close()
	if todayResp.StatusCode != http.StatusOK {
		t.Fatalf("today sessions status: %d", todayResp.StatusCode)
	}

	transferResp := authRequest(t, http.MethodPatch, srv.URL+"/api/v1/enrollments/"+enrollment.ID+"/transfer", token, map[string]string{
		"batch_id": batchB.ID,
	})
	defer transferResp.Body.Close()
	if transferResp.StatusCode != http.StatusOK {
		t.Fatalf("transfer status: %d", transferResp.StatusCode)
	}

	historyResp := authRequest(t, http.MethodGet, srv.URL+"/api/v1/students/"+student.ID+"/enrollments/history", token, nil)
	defer historyResp.Body.Close()
	if historyResp.StatusCode != http.StatusOK {
		t.Fatalf("history status: %d", historyResp.StatusCode)
	}
}

func TestStudentLoginForbiddenAdmin(t *testing.T) {
	srv := testDB(t)
	adminToken := login(t, srv, "test-admin@lms.local", "testpass123")

	studentResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/students", adminToken, map[string]string{
		"email": fmt.Sprintf("rbac-student-%d@lms.local", time.Now().UnixNano()), "password": "stud123", "name": "RBAC Student",
	})
	defer studentResp.Body.Close()
	var created struct{ Email string `json:"email"` }
	json.NewDecoder(studentResp.Body).Decode(&created)

	studentToken := login(t, srv, created.Email, "stud123")
	forbiddenResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/staff", studentToken, map[string]string{
		"email": "hacker@lms.local", "password": "x", "name": "Hacker",
	})
	defer forbiddenResp.Body.Close()
	if forbiddenResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", forbiddenResp.StatusCode)
	}
}
