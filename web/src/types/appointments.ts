export type ListResponse<T> = {
  items: T[];
};

export type Slot = {
  id: string;
  professional_id: string;
  start_time: string;
  end_time: string;
  status: 'available' | 'booked';
  created_at: string;
  updated_at: string;
};

export type Appointment = {
  id: string;
  slot_id: string;
  professional_id: string;
  patient_id: string;
  status: 'booked' | 'cancelled';
  created_at: string;
  updated_at: string;
  cancelled_at?: string | null;
};

export type ScheduleTemplateWindow = {
	start_time: string;
	end_time: string;
	slot_duration_minutes: 15 | 20 | 30 | 60;
};

export type ScheduleRecurrence = Partial<{
	monday: ScheduleTemplateWindow;
	tuesday: ScheduleTemplateWindow;
	wednesday: ScheduleTemplateWindow;
	thursday: ScheduleTemplateWindow;
	friday: ScheduleTemplateWindow;
	saturday: ScheduleTemplateWindow;
	sunday: ScheduleTemplateWindow;
}>;

export type ScheduleTemplateVersion = {
	id: string;
	template_id: string;
	version_number: number;
	effective_from: string;
	recurrence: ScheduleRecurrence;
	created_at: string;
	created_by?: string | null;
	reason?: string | null;
};

export type ScheduleTemplate = {
	id: string;
	professional_id: string;
	created_at: string;
	updated_at: string;
	versions?: ScheduleTemplateVersion[];
};

export type GetScheduleTemplateFilters = {
	professional_id?: string;
	effective_date: string;
};

export type ListScheduleTemplateVersionFilters = {
	template_id: string;
};

export type CreateScheduleTemplateVersionPayload = {
	professional_id: string;
	effective_from: string;
	recurrence: ScheduleRecurrence;
	reason?: string;
	created_by?: string | null;
};

export type ScheduleBlockScope = 'single' | 'range' | 'template';

export type ScheduleBlock = {
	id: string;
	professional_id: string;
	scope: ScheduleBlockScope;
	block_date?: string | null;
	start_date?: string | null;
	end_date?: string | null;
	day_of_week?: 1 | 2 | 3 | 4 | 5 | 6 | 7 | null;
	start_time: string;
	end_time: string;
	template_id?: string | null;
	created_at: string;
	updated_at: string;
};

export type ConsultationStatus = 'scheduled' | 'checked_in' | 'completed' | 'cancelled' | 'no_show';

export type ConsultationSource = 'online' | 'secretary' | 'doctor';

export type Consultation = {
	id: string;
	slot_id?: string | null;
	professional_id: string;
	patient_id: string;
	status: ConsultationStatus;
	source: ConsultationSource;
	scheduled_start: string;
	scheduled_end: string;
	notes?: string | null;
	check_in_time?: string | null;
	reception_notes?: string | null;
	created_at: string;
	updated_at: string;
	cancelled_at?: string | null;
};

export type WeekAgenda = {
	professional_id: string;
	week_start: string;
	templates: ScheduleTemplate[];
	blocks: ScheduleBlock[];
	consultations: Consultation[];
	slots: Slot[];
};

export type CreateAppointmentPayload = {
  slot_id: string;
  patient_id: string;
  professional_id: string;
};

export type BulkCreateSlotsPayload = {
  professional_id: string;
  date: string;
  start_time: string;
  end_time: string;
  slot_duration_minutes: 15 | 20 | 30 | 60;
};
