
-- machp / machp123

grant all on machp_dev.* to 'machp';

create database machp_dev;

use machp_dev;

drop table if exists tenant;
create table tenant (
 id smallint unsigned not null auto_increment
,name varchar(16) not null
,primary key (id)
,unique key (name)
) engine=innodb;

select * from tenant;

truncate table tenant;
