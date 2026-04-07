export type ListResponse<T> = {
  items: T[];
};

export type CreatePatientPayload = {
  first_name: string;
  last_name: string;
  document: string;
  birth_date: string;
  phone: string;
  email: string;
};

export type CreateProfessionalPayload = {
  first_name: string;
  last_name: string;
  specialty: string;
};

export type Patient = {
  id: string;
  first_name: string;
  last_name: string;
  document: string;
  birth_date: string;
  phone: string;
  email?: string | null;
  active: boolean;
  created_at: string;
  updated_at: string;
};

export type Professional = {
  id: string;
  first_name: string;
  last_name: string;
  specialty: string;
  active: boolean;
  created_at: string;
  updated_at: string;
};
