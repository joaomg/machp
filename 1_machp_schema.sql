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
,`md5` char(32) not null
,name varchar(16) not null
,primary key (id)
,unique key (md5)
,unique key (name)
) engine=innodb;
