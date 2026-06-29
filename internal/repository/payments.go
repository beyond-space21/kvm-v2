package repository

import (
	"context"
	"database/sql"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) GetInvoiceByEnrollment(ctx context.Context, enrollmentID string) (*models.FeeInvoice, error) {
	return r.scanInvoice(r.db.QueryRowContext(ctx, `
		SELECT fi.id, fi.enrollment_id, fi.amount, fi.currency, fi.status,
		       fi.due_date::text, COALESCE(SUM(fp.amount), 0), fi.created_at, fi.updated_at
		FROM fee_invoices fi
		LEFT JOIN fee_payments fp ON fp.invoice_id = fi.id
		WHERE fi.enrollment_id = $1
		GROUP BY fi.id
	`, enrollmentID))
}

func (r *PaymentRepository) ListPayments(ctx context.Context, enrollmentID string, offset, limit int) ([]models.FeePayment, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM fee_payments fp
		JOIN fee_invoices fi ON fi.id = fp.invoice_id
		WHERE fi.enrollment_id = $1
	`, enrollmentID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT fp.id, fp.invoice_id, fp.amount, fp.paid_at, fp.method, fp.reference, fp.recorded_by, fp.created_at
		FROM fee_payments fp
		JOIN fee_invoices fi ON fi.id = fp.invoice_id
		WHERE fi.enrollment_id = $1
		ORDER BY fp.paid_at DESC
		LIMIT $2 OFFSET $3
	`, enrollmentID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payments []models.FeePayment
	for rows.Next() {
		p, err := r.scanPaymentRow(rows)
		if err != nil {
			return nil, 0, err
		}
		payments = append(payments, *p)
	}
	return payments, total, rows.Err()
}

func (r *PaymentRepository) RecordPayment(ctx context.Context, enrollmentID string, amount float64, method, reference *string, recordedBy string) (*models.FeePayment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var invoiceID string
	var invoiceAmount float64
	var paidSoFar float64
	err = tx.QueryRowContext(ctx, `
		SELECT fi.id, fi.amount,
		       (SELECT COALESCE(SUM(fp.amount), 0) FROM fee_payments fp WHERE fp.invoice_id = fi.id)
		FROM fee_invoices fi
		WHERE fi.enrollment_id = $1
		FOR UPDATE
	`, enrollmentID).Scan(&invoiceID, &invoiceAmount, &paidSoFar)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("invoice not found for enrollment")
	}
	if err != nil {
		return nil, err
	}

	var methodVal, refVal sql.NullString
	if method != nil {
		methodVal = sql.NullString{String: *method, Valid: true}
	}
	if reference != nil {
		refVal = sql.NullString{String: *reference, Valid: true}
	}

	var p models.FeePayment
	err = tx.QueryRowContext(ctx, `
		INSERT INTO fee_payments (invoice_id, amount, method, reference, recorded_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, invoice_id, amount, paid_at, method, reference, recorded_by, created_at
	`, invoiceID, amount, methodVal, refVal, recordedBy).Scan(
		&p.ID, &p.InvoiceID, &p.Amount, &p.PaidAt, &methodVal, &refVal, &p.RecordedBy, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if methodVal.Valid {
		p.Method = &methodVal.String
	}
	if refVal.Valid {
		p.Reference = &refVal.String
	}

	totalPaid := paidSoFar + amount
	status := models.InvoicePartial
	if totalPaid >= invoiceAmount {
		status = models.InvoicePaid
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE fee_invoices SET status = $2, updated_at = NOW() WHERE id = $1
	`, invoiceID, status); err != nil {
		return nil, err
	}

	return &p, tx.Commit()
}

func (r *PaymentRepository) scanInvoice(row *sql.Row) (*models.FeeInvoice, error) {
	var inv models.FeeInvoice
	var dueDate sql.NullString
	err := row.Scan(&inv.ID, &inv.EnrollmentID, &inv.Amount, &inv.Currency, &inv.Status,
		&dueDate, &inv.PaidAmount, &inv.CreatedAt, &inv.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, httpx.NotFound("invoice not found")
	}
	if err != nil {
		return nil, err
	}
	if dueDate.Valid {
		inv.DueDate = &dueDate.String
	}
	return &inv, nil
}

func (r *PaymentRepository) scanPaymentRow(rows *sql.Rows) (*models.FeePayment, error) {
	var p models.FeePayment
	var method, ref sql.NullString
	err := rows.Scan(&p.ID, &p.InvoiceID, &p.Amount, &p.PaidAt, &method, &ref, &p.RecordedBy, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	if method.Valid {
		p.Method = &method.String
	}
	if ref.Valid {
		p.Reference = &ref.String
	}
	return &p, nil
}
