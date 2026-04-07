package directory

import (
	"context"
	"testing"
	"time"
)

func TestRepositoryIntegrationCreateEncounterImplicitlyCreatesChartAndInitialNote(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	patient := seedClinicalPatient(t, repo, "encounter-create")
	professional := seedClinicalProfessional(t, repo, "encounter-create")
	occurredAt := time.Date(2026, 4, 7, 14, 30, 0, 0, time.UTC)

	encounter, err := repo.CreateEncounter(context.Background(), CreateEncounterParams{
		PatientID:      patient.ID,
		ProfessionalID: professional.ID,
		OccurredAt:     occurredAt.Format(time.RFC3339),
		Note:           "Paciente estable y orientado.",
	})
	if err != nil {
		t.Fatalf("create encounter: %v", err)
	}

	if got := countClinicalCharts(t, db); got != 1 {
		t.Fatalf("clinical charts persisted = %d, want 1", got)
	}
	if got := countClinicalEncounters(t, db); got != 1 {
		t.Fatalf("clinical encounters persisted = %d, want 1", got)
	}
	if got := countClinicalNotes(t, db); got != 1 {
		t.Fatalf("clinical notes persisted = %d, want 1", got)
	}

	chart := fetchClinicalChartByOwner(t, db, patient.ID, professional.ID)
	if encounter.ChartID != chart.ID {
		t.Fatalf("encounter chart_id = %q, want %q", encounter.ChartID, chart.ID)
	}

	persistedEncounter := fetchEncounterByID(t, db, encounter.ID)
	if persistedEncounter.ChartID != chart.ID {
		t.Fatalf("persisted encounter chart_id = %q, want %q", persistedEncounter.ChartID, chart.ID)
	}
	if !persistedEncounter.OccurredAt.Equal(occurredAt) {
		t.Fatalf("persisted encounter occurred_at = %s, want %s", persistedEncounter.OccurredAt, occurredAt)
	}

	persistedNote := fetchClinicalNoteByEncounterID(t, db, encounter.ID)
	if persistedNote.ChartID != chart.ID {
		t.Fatalf("persisted note chart_id = %q, want %q", persistedNote.ChartID, chart.ID)
	}
	if persistedNote.Content != "Paciente estable y orientado." {
		t.Fatalf("persisted note content = %q, want %q", persistedNote.Content, "Paciente estable y orientado.")
	}
	if persistedNote.Kind != "initial" {
		t.Fatalf("persisted note kind = %q, want initial", persistedNote.Kind)
	}

	if encounter.InitialNote.ID != persistedNote.ID {
		t.Fatalf("returned initial note id = %q, want %q", encounter.InitialNote.ID, persistedNote.ID)
	}
	if encounter.InitialNote.Content != persistedNote.Content {
		t.Fatalf("returned initial note content = %q, want %q", encounter.InitialNote.Content, persistedNote.Content)
	}
}

func TestRepositoryIntegrationListPatientEncountersScopesByPatientAndProfessional(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	patient := seedClinicalPatient(t, repo, "list-owned")
	otherPatient := seedClinicalPatient(t, repo, "list-other-patient")
	professional := seedClinicalProfessional(t, repo, "list-owned")

	olderEncounter, err := repo.CreateEncounter(context.Background(), CreateEncounterParams{
		PatientID:      patient.ID,
		ProfessionalID: professional.ID,
		OccurredAt:     time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Note:           "Control inicial.",
	})
	if err != nil {
		t.Fatalf("create older encounter: %v", err)
	}

	newerEncounter, err := repo.CreateEncounter(context.Background(), CreateEncounterParams{
		PatientID:      patient.ID,
		ProfessionalID: professional.ID,
		OccurredAt:     time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Note:           "Control evolutivo.",
	})
	if err != nil {
		t.Fatalf("create newer encounter: %v", err)
	}

	_, err = repo.CreateEncounter(context.Background(), CreateEncounterParams{
		PatientID:      otherPatient.ID,
		ProfessionalID: professional.ID,
		OccurredAt:     time.Date(2026, 4, 7, 13, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Note:           "Otro paciente.",
	})
	if err != nil {
		t.Fatalf("create other patient encounter: %v", err)
	}

	if got := countClinicalCharts(t, db); got != 2 {
		t.Fatalf("clinical charts persisted = %d, want 2", got)
	}

	encounters, err := repo.ListPatientEncounters(context.Background(), patient.ID, professional.ID)
	if err != nil {
		t.Fatalf("list patient encounters: %v", err)
	}

	if len(encounters) != 2 {
		t.Fatalf("encounters listed = %d, want 2", len(encounters))
	}

	if encounters[0].ID != newerEncounter.ID {
		t.Fatalf("first encounter id = %q, want %q", encounters[0].ID, newerEncounter.ID)
	}
	if encounters[1].ID != olderEncounter.ID {
		t.Fatalf("second encounter id = %q, want %q", encounters[1].ID, olderEncounter.ID)
	}

	for _, encounter := range encounters {
		if encounter.PatientID != patient.ID {
			t.Fatalf("encounter patient_id = %q, want %q", encounter.PatientID, patient.ID)
		}
		if encounter.ProfessionalID != professional.ID {
			t.Fatalf("encounter professional_id = %q, want %q", encounter.ProfessionalID, professional.ID)
		}
		if encounter.InitialNote.Kind != "initial" {
			t.Fatalf("encounter note kind = %q, want initial", encounter.InitialNote.Kind)
		}
	}
}

func TestRepositoryIntegrationProfessionalCannotSeeOtherProfessionalsEncounters(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	patient := seedClinicalPatient(t, repo, "professional-scope")
	ownerProfessional := seedClinicalProfessional(t, repo, "professional-owner")
	otherProfessional := seedClinicalProfessional(t, repo, "professional-other")

	ownerEncounter, err := repo.CreateEncounter(context.Background(), CreateEncounterParams{
		PatientID:      patient.ID,
		ProfessionalID: ownerProfessional.ID,
		OccurredAt:     time.Date(2026, 4, 7, 9, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Note:           "Consulta del profesional titular.",
	})
	if err != nil {
		t.Fatalf("create owner encounter: %v", err)
	}

	otherEncounter, err := repo.CreateEncounter(context.Background(), CreateEncounterParams{
		PatientID:      patient.ID,
		ProfessionalID: otherProfessional.ID,
		OccurredAt:     time.Date(2026, 4, 7, 11, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Note:           "Consulta de otro profesional.",
	})
	if err != nil {
		t.Fatalf("create other professional encounter: %v", err)
	}

	if got := countClinicalCharts(t, db); got != 2 {
		t.Fatalf("clinical charts persisted = %d, want 2", got)
	}

	ownerView, err := repo.ListPatientEncounters(context.Background(), patient.ID, ownerProfessional.ID)
	if err != nil {
		t.Fatalf("list owner encounters: %v", err)
	}
	if len(ownerView) != 1 {
		t.Fatalf("owner encounters listed = %d, want 1", len(ownerView))
	}
	if ownerView[0].ID != ownerEncounter.ID {
		t.Fatalf("owner encounter id = %q, want %q", ownerView[0].ID, ownerEncounter.ID)
	}

	otherView, err := repo.ListPatientEncounters(context.Background(), patient.ID, otherProfessional.ID)
	if err != nil {
		t.Fatalf("list other professional encounters: %v", err)
	}
	if len(otherView) != 1 {
		t.Fatalf("other professional encounters listed = %d, want 1", len(otherView))
	}
	if otherView[0].ID != otherEncounter.ID {
		t.Fatalf("other professional encounter id = %q, want %q", otherView[0].ID, otherEncounter.ID)
	}
	if otherView[0].ID == ownerEncounter.ID {
		t.Fatalf("other professional should not see encounter %q", ownerEncounter.ID)
	}
}
