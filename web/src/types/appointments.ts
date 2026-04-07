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
