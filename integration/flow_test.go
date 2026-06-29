package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func classByGrade(t *testing.T, srvURL, token string, grade int) string {
	t.Helper()
	listResp := authRequest(t, http.MethodGet, srvURL+"/api/v1/classes", token, nil)
	defer listResp.Body.Close()
	var list struct {
		Data []struct {
			ID    string `json:"id"`
			Grade int    `json:"grade"`
		} `json:"data"`
	}
	json.NewDecoder(listResp.Body).Decode(&list)
	for _, c := range list.Data {
		if c.Grade == grade {
			return c.ID
		}
	}
	resp := authRequest(t, http.MethodPost, srvURL+"/api/v1/classes", token, map[string]any{
		"name": fmt.Sprintf("Class %d", grade), "grade": grade,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create class %d status: %d", grade, resp.StatusCode)
	}
	var class struct{ ID string `json:"id"` }
	json.NewDecoder(resp.Body).Decode(&class)
	return class.ID
}

func subjectByCode(t *testing.T, srvURL, token, name, code string) string {
	t.Helper()
	listResp := authRequest(t, http.MethodGet, srvURL+"/api/v1/subjects", token, nil)
	defer listResp.Body.Close()
	var list struct {
		Data []struct {
			ID   string `json:"id"`
			Code string `json:"code"`
		} `json:"data"`
	}
	json.NewDecoder(listResp.Body).Decode(&list)
	for _, s := range list.Data {
		if s.Code == code {
			return s.ID
		}
	}
	resp := authRequest(t, http.MethodPost, srvURL+"/api/v1/subjects", token, map[string]string{
		"name": name, "code": code,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create subject %s status: %d", code, resp.StatusCode)
	}
	var subject struct{ ID string `json:"id"` }
	json.NewDecoder(resp.Body).Decode(&subject)
	return subject.ID
}

func offeringByClassSubject(t *testing.T, srvURL, token, classID, subjectID string, fee float64) string {
	t.Helper()
	listResp := authRequest(t, http.MethodGet, srvURL+fmt.Sprintf("/api/v1/classes/%s/offerings", classID), token, nil)
	defer listResp.Body.Close()
	var list struct {
		Data []struct {
			ID        string `json:"id"`
			SubjectID string `json:"subject_id"`
		} `json:"data"`
	}
	json.NewDecoder(listResp.Body).Decode(&list)
	for _, o := range list.Data {
		if o.SubjectID == subjectID {
			return o.ID
		}
	}
	resp := authRequest(t, http.MethodPost, srvURL+"/api/v1/offerings", token, map[string]any{
		"class_id": classID, "subject_id": subjectID, "fee_amount": fee,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create offering status: %d", resp.StatusCode)
	}
	var offering struct{ ID string `json:"id"` }
	json.NewDecoder(resp.Body).Decode(&offering)
	return offering.ID
}

func TestEnrollmentAttendanceTransfer(t *testing.T) {
	srv := testDB(t)
	token := login(t, srv, "test-admin@lms.local", "testpass123")

	class11ID := classByGrade(t, srv.URL, token, 11)
	physicsID := subjectByCode(t, srv.URL, token, "Physics", "PHY")

	yearResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/academic-years", token, map[string]any{
		"name": fmt.Sprintf("flow-year-%d", time.Now().UnixNano()), "start_date": "2026-04-01", "end_date": "2027-03-31", "is_active": true,
	})
	defer yearResp.Body.Close()
	var year struct{ ID string `json:"id"` }
	json.NewDecoder(yearResp.Body).Decode(&year)

	offeringID := offeringByClassSubject(t, srv.URL, token, class11ID, physicsID, 5000)

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
		"email": fmt.Sprintf("flow-student-%d@lms.local", time.Now().UnixNano()), "password": "stud123", "name": "Flow Student", "class_id": class11ID,
	})
	defer studentResp.Body.Close()
	if studentResp.StatusCode != http.StatusCreated {
		t.Fatalf("create student status: %d", studentResp.StatusCode)
	}
	var student struct{ ID string `json:"id"` }
	json.NewDecoder(studentResp.Body).Decode(&student)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	batchAResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/batches", token, map[string]any{
		"offering_id": offeringID, "name": "Batch A-" + suffix, "teacher_id": staff.ID,
	})
	defer batchAResp.Body.Close()
	if batchAResp.StatusCode != http.StatusCreated {
		t.Fatalf("create batch A status: %d", batchAResp.StatusCode)
	}
	var batchA struct{ ID string `json:"id"` }
	json.NewDecoder(batchAResp.Body).Decode(&batchA)

	batchBResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/batches", token, map[string]any{
		"offering_id": offeringID, "name": "Batch B-" + suffix, "teacher_id": staff.ID,
	})
	defer batchBResp.Body.Close()
	if batchBResp.StatusCode != http.StatusCreated {
		t.Fatalf("create batch B status: %d", batchBResp.StatusCode)
	}
	var batchB struct{ ID string `json:"id"` }
	json.NewDecoder(batchBResp.Body).Decode(&batchB)

	enrollResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/enrollments", token, map[string]string{
		"student_id": student.ID, "academic_year_id": year.ID, "offering_id": offeringID, "batch_id": batchA.ID,
	})
	defer enrollResp.Body.Close()
	if enrollResp.StatusCode != http.StatusCreated {
		t.Fatalf("enroll status: %d", enrollResp.StatusCode)
	}
	var enrollment struct{ ID string `json:"id"` }
	json.NewDecoder(enrollResp.Body).Decode(&enrollment)

	invoiceResp := authRequest(t, http.MethodGet, srv.URL+"/api/v1/enrollments/"+enrollment.ID+"/invoice", token, nil)
	defer invoiceResp.Body.Close()
	if invoiceResp.StatusCode != http.StatusOK {
		t.Fatalf("invoice status: %d", invoiceResp.StatusCode)
	}

	paymentResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/enrollments/"+enrollment.ID+"/payments", token, map[string]any{
		"amount": 5000, "method": "cash",
	})
	defer paymentResp.Body.Close()
	if paymentResp.StatusCode != http.StatusCreated {
		t.Fatalf("payment status: %d", paymentResp.StatusCode)
	}

	tmplResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/session-templates", token, map[string]any{
		"batch_id": batchA.ID, "day_of_week": 1, "start_time": "17:00", "end_time": "18:00",
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
	classID := classByGrade(t, srv.URL, adminToken, 10)

	studentResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/students", adminToken, map[string]string{
		"email": fmt.Sprintf("rbac-student-%d@lms.local", time.Now().UnixNano()), "password": "stud123", "name": "RBAC Student", "class_id": classID,
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

func TestEnrollmentClassMismatch(t *testing.T) {
	srv := testDB(t)
	token := login(t, srv, "test-admin@lms.local", "testpass123")

	class10ID := classByGrade(t, srv.URL, token, 10)
	class11ID := classByGrade(t, srv.URL, token, 11)
	physicsID := subjectByCode(t, srv.URL, token, "Physics", "PHY")

	yearResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/academic-years", token, map[string]any{
		"name": fmt.Sprintf("mismatch-year-%d", time.Now().UnixNano()), "start_date": "2026-04-01", "end_date": "2027-03-31", "is_active": true,
	})
	defer yearResp.Body.Close()
	var year struct{ ID string `json:"id"` }
	json.NewDecoder(yearResp.Body).Decode(&year)

	offeringID := offeringByClassSubject(t, srv.URL, token, class11ID, physicsID, 5000)

	staffResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/staff", token, map[string]string{
		"email": fmt.Sprintf("mismatch-teacher-%d@lms.local", time.Now().UnixNano()), "password": "teach123", "name": "Teacher",
	})
	defer staffResp.Body.Close()
	var staff struct{ ID string `json:"id"` }
	json.NewDecoder(staffResp.Body).Decode(&staff)

	studentResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/students", token, map[string]string{
		"email": fmt.Sprintf("mismatch-student-%d@lms.local", time.Now().UnixNano()), "password": "stud123", "name": "Student", "class_id": class10ID,
	})
	defer studentResp.Body.Close()
	var student struct{ ID string `json:"id"` }
	json.NewDecoder(studentResp.Body).Decode(&student)

	batchResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/batches", token, map[string]any{
		"offering_id": offeringID, "name": "Mismatch Batch-" + fmt.Sprintf("%d", time.Now().UnixNano()), "teacher_id": staff.ID,
	})
	defer batchResp.Body.Close()
	if batchResp.StatusCode != http.StatusCreated {
		t.Fatalf("create batch status: %d", batchResp.StatusCode)
	}
	var batch struct{ ID string `json:"id"` }
	json.NewDecoder(batchResp.Body).Decode(&batch)

	enrollResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/enrollments", token, map[string]string{
		"student_id": student.ID, "academic_year_id": year.ID, "offering_id": offeringID, "batch_id": batch.ID,
	})
	defer enrollResp.Body.Close()
	if enrollResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for class mismatch, got %d", enrollResp.StatusCode)
	}
}
