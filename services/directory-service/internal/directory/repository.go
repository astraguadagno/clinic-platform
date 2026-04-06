package directory

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var ErrNotFound = errors.New("directory resource not found")

type Repository struct {
	db *sql.DB
}

type CreatePatientParams struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Document  string `json:"document"`
	BirthDate string `json:"birth_date"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
}

type Patient struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Document  string    `json:"document"`
	BirthDate string    `json:"birth_date"`
	Phone     string    `json:"phone"`
	Email     *string   `json:"email,omitempty"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateProfessionalParams struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Specialty string `json:"specialty"`
}

type Professional struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Specialty string    `json:"specialty"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func (r *Repository) CreatePatient(ctx context.Context, params CreatePatientParams) (Patient, error) {
	birthDate, err := time.Parse("2006-01-02", strings.TrimSpace(params.BirthDate))
	if err != nil {
		return Patient{}, err
	}

	query := `
		INSERT INTO patients (first_name, last_name, document, birth_date, phone, email)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
		RETURNING id, first_name, last_name, document, birth_date, phone, email, active, created_at, updated_at
	`

	row := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(params.FirstName),
		strings.TrimSpace(params.LastName),
		strings.TrimSpace(params.Document),
		birthDate,
		strings.TrimSpace(params.Phone),
		strings.TrimSpace(params.Email),
	)

	return scanPatient(row)
}

func (r *Repository) ListPatients(ctx context.Context) ([]Patient, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, first_name, last_name, document, birth_date, phone, email, active, created_at, updated_at
		FROM patients
		ORDER BY last_name, first_name, created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	patients := make([]Patient, 0)
	for rows.Next() {
		patient, err := scanPatient(rows)
		if err != nil {
			return nil, err
		}
		patients = append(patients, patient)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return patients, nil
}

func (r *Repository) GetPatientByID(ctx context.Context, id string) (Patient, error) {
	if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
		return Patient{}, ErrNotFound
	}

	query := `
		SELECT id, first_name, last_name, document, birth_date, phone, email, active, created_at, updated_at
		FROM patients
		WHERE id = $1
	`

	patient, err := scanPatient(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Patient{}, ErrNotFound
	}

	return patient, err
}

func (r *Repository) CreateProfessional(ctx context.Context, params CreateProfessionalParams) (Professional, error) {
	query := `
		INSERT INTO professionals (first_name, last_name, specialty)
		VALUES ($1, $2, $3)
		RETURNING id, first_name, last_name, specialty, active, created_at, updated_at
	`

	row := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(params.FirstName),
		strings.TrimSpace(params.LastName),
		strings.TrimSpace(params.Specialty),
	)

	return scanProfessional(row)
}

func (r *Repository) ListProfessionals(ctx context.Context) ([]Professional, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, first_name, last_name, specialty, active, created_at, updated_at
		FROM professionals
		ORDER BY last_name, first_name, created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	professionals := make([]Professional, 0)
	for rows.Next() {
		professional, err := scanProfessional(rows)
		if err != nil {
			return nil, err
		}
		professionals = append(professionals, professional)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return professionals, nil
}

func (r *Repository) GetProfessionalByID(ctx context.Context, id string) (Professional, error) {
	if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
		return Professional{}, ErrNotFound
	}

	query := `
		SELECT id, first_name, last_name, specialty, active, created_at, updated_at
		FROM professionals
		WHERE id = $1
	`

	professional, err := scanProfessional(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Professional{}, ErrNotFound
	}

	return professional, err
}

type patientScanner interface {
	Scan(dest ...any) error
}

func scanPatient(scanner patientScanner) (Patient, error) {
	var patient Patient
	var birthDate time.Time
	var email sql.NullString

	err := scanner.Scan(
		&patient.ID,
		&patient.FirstName,
		&patient.LastName,
		&patient.Document,
		&birthDate,
		&patient.Phone,
		&email,
		&patient.Active,
		&patient.CreatedAt,
		&patient.UpdatedAt,
	)
	if err != nil {
		return Patient{}, err
	}

	patient.BirthDate = birthDate.Format("2006-01-02")
	if email.Valid {
		patient.Email = &email.String
	}

	return patient, nil
}

type professionalScanner interface {
	Scan(dest ...any) error
}

func scanProfessional(scanner professionalScanner) (Professional, error) {
	var professional Professional

	err := scanner.Scan(
		&professional.ID,
		&professional.FirstName,
		&professional.LastName,
		&professional.Specialty,
		&professional.Active,
		&professional.CreatedAt,
		&professional.UpdatedAt,
	)
	if err != nil {
		return Professional{}, err
	}

	return professional, nil
}
