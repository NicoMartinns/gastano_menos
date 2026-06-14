ALTER TABLE categories DROP CONSTRAINT fk_categories_parent;
ALTER TABLE categories DROP COLUMN parent_id;