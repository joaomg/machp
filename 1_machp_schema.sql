-- drop schema if it exists 
drop schema if exists machp_dev;
-- create empty schema
create schema machp_dev;

-- move session to schema
use machp_dev;

-- create schema tables
drop table if exists tenant;
create table tenant (
 id smallint unsigned not null auto_increment
,name varchar(16) not null
,primary key (id)
,unique key (name)
) engine=innodb;
