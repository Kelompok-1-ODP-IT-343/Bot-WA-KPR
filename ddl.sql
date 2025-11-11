-- public.kpr_rates definition

-- Drop table

-- DROP TABLE kpr_rates;

CREATE TABLE kpr_rates (
	id serial4 NOT NULL,
	rate_name varchar(100) NOT NULL,
	rate_type public.rate_type NOT NULL,
	property_type public.kpr_property_type NOT NULL,
	customer_segment public.customer_segment NOT NULL,
	base_rate numeric(5, 4) NOT NULL,
	margin numeric(5, 4) NOT NULL,
	effective_rate numeric(5, 4) NOT NULL,
	min_loan_amount numeric(15, 2) NOT NULL,
	max_loan_amount numeric(15, 2) NOT NULL,
	min_term_years int4 NOT NULL,
	max_term_years int4 NOT NULL,
	max_ltv_ratio numeric(5, 4) NOT NULL,
	min_income numeric(15, 2) NOT NULL,
	max_age int4 NOT NULL,
	min_down_payment_percent numeric(5, 2) NOT NULL,
	admin_fee numeric(15, 2) DEFAULT 0 NULL,
	admin_fee_percent numeric(5, 4) DEFAULT 0 NULL,
	appraisal_fee numeric(15, 2) DEFAULT 0 NULL,
	insurance_rate numeric(5, 4) DEFAULT 0 NULL,
	notary_fee_percent numeric(5, 4) DEFAULT 0 NULL,
	is_promotional bool DEFAULT false NULL,
	promo_description text NULL,
	promo_start_date date NULL,
	promo_end_date date NULL,
	is_active bool DEFAULT true NULL,
	effective_date date NOT NULL,
	expiry_date date NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT kpr_rates_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE public.kpr_rates IS 'KPR interest rates and loan terms configuration';


-- public.roles definition

-- Drop table

-- DROP TABLE roles;

CREATE TABLE roles (
	id serial4 NOT NULL,
	"name" varchar(50) NOT NULL,
	description text NULL,
	permissions json NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT roles_name_key UNIQUE (name),
	CONSTRAINT roles_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_roles_name ON public.roles USING btree (name);
COMMENT ON TABLE public.roles IS 'Role-based access control (RBAC) for system security';


-- public.users definition

-- Drop table

-- DROP TABLE users;

CREATE TABLE users (
	id serial4 NOT NULL,
	username varchar(50) NOT NULL,
	email varchar(100) NOT NULL,
	phone varchar(20) NULL,
	password_hash varchar(255) NOT NULL,
	role_id int4 NOT NULL,
	status public.user_status DEFAULT 'PENDING_VERIFICATION'::user_status NULL,
	email_verified_at timestamp NULL,
	phone_verified_at timestamp NULL,
	last_login_at timestamp NULL,
	failed_login_attempts int4 DEFAULT 0 NULL,
	locked_until timestamp NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	consent_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT users_email_key UNIQUE (email),
	CONSTRAINT users_phone_key UNIQUE (phone),
	CONSTRAINT users_pkey PRIMARY KEY (id),
	CONSTRAINT users_username_key UNIQUE (username),
	CONSTRAINT users_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(id)
);
COMMENT ON TABLE public.users IS 'Core user table with enhanced security features';


-- public.branch_staff definition

-- Drop table

-- DROP TABLE branch_staff;

CREATE TABLE branch_staff (
	id serial4 NOT NULL,
	user_id int4 NOT NULL,
	branch_code varchar(10) NOT NULL,
	staff_id varchar(20) NOT NULL,
	"position" public.staff_position NOT NULL,
	supervisor_id int4 NULL,
	is_active bool DEFAULT true NULL,
	start_date date NOT NULL,
	end_date date NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT branch_staff_pkey PRIMARY KEY (id),
	CONSTRAINT branch_staff_staff_id_key UNIQUE (staff_id),
	CONSTRAINT branch_staff_supervisor_id_fkey FOREIGN KEY (supervisor_id) REFERENCES branch_staff(id),
	CONSTRAINT branch_staff_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id)
);
COMMENT ON TABLE public.branch_staff IS 'Branch staff hierarchy and positions';


-- public.user_profiles definition

-- Drop table

-- DROP TABLE user_profiles;

CREATE TABLE user_profiles (
	id serial4 NOT NULL,
	user_id int4 NOT NULL,
	full_name varchar(100) NOT NULL,
	nik varchar(16) NULL,
	npwp varchar(16) NULL,
	birth_date date NOT NULL,
	birth_place varchar(100) NOT NULL,
	gender public.gender_type NULL,
	marital_status public.marital_status_type NULL,
	address text NULL,
	city varchar(100) NULL,
	province varchar(100) NULL,
	postal_code varchar(10) NULL,
	occupation varchar(100) NOT NULL,
	company_name varchar(100) NULL,
	monthly_income numeric(15, 2) NOT NULL,
	work_experience int4 DEFAULT 0 NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	district varchar(255) NULL,
	sub_district varchar(255) NULL,
	company_address varchar(255) NULL,
	company_city varchar(100) NULL,
	company_province varchar(100) NULL,
	company_postal_code varchar(10) NULL,
	company_district varchar(255) NULL,
	company_subdistrict varchar(255) NULL,
	CONSTRAINT user_profiles_nik_key UNIQUE (nik),
	CONSTRAINT user_profiles_npwp_key UNIQUE (npwp),
	CONSTRAINT user_profiles_pkey PRIMARY KEY (id),
	CONSTRAINT user_profiles_user_id_key UNIQUE (user_id),
	CONSTRAINT user_profiles_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
COMMENT ON TABLE public.user_profiles IS 'Extended user profile with KYC (Know Your Customer) data';


-- public.approval_workflow definition

-- Drop table

-- DROP TABLE approval_workflow;

CREATE TABLE approval_workflow (
	id serial4 NOT NULL,
	application_id int4 NOT NULL,
	stage public.workflow_stage NOT NULL,
	assigned_to int4 NOT NULL,
	status public.workflow_status DEFAULT 'PENDING'::workflow_status NULL,
	priority public.priority_level DEFAULT 'NORMAL'::priority_level NULL,
	due_date timestamp NULL,
	started_at timestamp NULL,
	completed_at timestamp NULL,
	approval_notes text NULL,
	rejection_reason text NULL,
	escalated_to int4 NULL,
	escalated_at timestamp NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT approval_workflow_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE public.approval_workflow IS 'Enhanced approval workflow with hierarchical levels';


-- public.kpr_applications definition

-- Drop table

-- DROP TABLE kpr_applications;

CREATE TABLE kpr_applications (
	id serial4 NOT NULL,
	application_number varchar(20) NOT NULL,
	user_id int4 NOT NULL,
	property_id int4 NULL,
	kpr_rate_id int4 NOT NULL,
	property_type public.application_property_type NOT NULL,
	property_value numeric(15, 2) NOT NULL,
	loan_amount numeric(15, 2) NOT NULL,
	loan_term_years int4 NOT NULL,
	interest_rate numeric(5, 4) NOT NULL,
	monthly_installment numeric(15, 2) NOT NULL,
	down_payment numeric(15, 2) NOT NULL,
	property_address text NOT NULL,
	property_certificate_type public.property_certificate_type NOT NULL,
	developer_name varchar(100) NULL,
	purpose public.application_purpose NOT NULL,
	status public.application_status DEFAULT 'DRAFT'::application_status NULL,
	submitted_at timestamp NULL,
	reviewed_at timestamp NULL,
	approved_at timestamp NULL,
	rejected_at timestamp NULL,
	rejection_reason text NULL,
	notes text NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	ltv_ratio numeric(5, 4) DEFAULT 0.0000 NOT NULL, -- Loan to Value ratio calculated as (loan_amount / property_value)
	CONSTRAINT kpr_applications_application_number_key UNIQUE (application_number),
	CONSTRAINT kpr_applications_pkey PRIMARY KEY (id)
);
COMMENT ON TABLE public.kpr_applications IS 'Main KPR application table with property and rate linking';

-- Column comments

COMMENT ON COLUMN public.kpr_applications.ltv_ratio IS 'Loan to Value ratio calculated as (loan_amount / property_value)';


-- public.approval_workflow foreign keys

ALTER TABLE public.approval_workflow ADD CONSTRAINT approval_workflow_application_id_fkey FOREIGN KEY (application_id) REFERENCES kpr_applications(id);
ALTER TABLE public.approval_workflow ADD CONSTRAINT approval_workflow_assigned_to_fkey FOREIGN KEY (assigned_to) REFERENCES users(id);
ALTER TABLE public.approval_workflow ADD CONSTRAINT approval_workflow_escalated_to_fkey FOREIGN KEY (escalated_to) REFERENCES users(id);


-- public.kpr_applications foreign keys

ALTER TABLE public.kpr_applications ADD CONSTRAINT kpr_applications_kpr_rate_id_fkey FOREIGN KEY (kpr_rate_id) REFERENCES kpr_rates(id);
ALTER TABLE public.kpr_applications ADD CONSTRAINT kpr_applications_property_id_fkey FOREIGN KEY (property_id) REFERENCES properties(id);
ALTER TABLE public.kpr_applications ADD CONSTRAINT kpr_applications_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);
