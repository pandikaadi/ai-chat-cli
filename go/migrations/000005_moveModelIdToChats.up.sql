ALTER TABLE models DROP COLUMN model_id;
ALTER TABLE chats ADD COLUMN model_id SERIAL;
ALTER TABLE chats ADD CONSTRAINT fk_model_id FOREIGN KEY (model_id) REFERENCES models(id);
