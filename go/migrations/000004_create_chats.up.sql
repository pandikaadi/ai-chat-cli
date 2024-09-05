CREATE TABLE chats ( 
    id BIGSERIAL PRIMARY KEY, 
    content TEXT NOT NULL, 
    topic_id BIGSERIAL, 
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
);
