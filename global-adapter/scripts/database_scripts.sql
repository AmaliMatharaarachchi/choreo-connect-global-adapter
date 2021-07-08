CREATE TABLE globalAdapter.dbo.ga_local_adapter_partition (
		api_uuid varchar(150) NOT NULL,
		label_hierarchy varchar(50) NOT NULL,
		api_id int NOT NULL,
		org_id varchar(150) NULL,
	CONSTRAINT PK_uuid_hierarcy PRIMARY KEY (api_uuid,label_hierarchy)
);

CREATE TABLE globalAdapter.dbo.la_partition_size (
		parition_size int NULL
);
