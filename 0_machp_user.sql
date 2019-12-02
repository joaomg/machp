-- create schema
create schema if not exists machp;

-- create user 
create user if not exists machp_dev identified by 'machp_dev123';

-- grant user full access to schema
grant all on machp_dev.* to 'machp';
