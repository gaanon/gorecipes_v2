-- Recipe Manager Database Schema for PostgreSQL
-- Production-ready schema with proper constraints, indexes, and data types

-- Enable UUID extension for generating unique IDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create ENUM types for better data consistency
CREATE TYPE measurement_system AS ENUM ('metric', 'imperial');

-- Main recipes table
CREATE TABLE recipes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    photo_filename VARCHAR(255), -- Local filename for stored photos
    serves INTEGER CHECK (serves > 0),
    prep_time_minutes INTEGER CHECK (prep_time_minutes >= 0),
    cook_time_minutes INTEGER CHECK (cook_time_minutes >= 0),
    total_time_minutes INTEGER GENERATED ALWAYS AS (prep_time_minutes + cook_time_minutes) STORED,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID, -- For multi-user systems
    
    -- Full-text search vector for efficient searching
    search_vector TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('english', COALESCE(title, '') || ' ' || COALESCE(description, ''))
    ) STORED
);

-- Ingredients master table (normalized approach for consistency)
CREATE TABLE ingredients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    category VARCHAR(100), -- e.g., 'dairy', 'vegetables', 'spices'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Units of measurement table
CREATE TABLE measurement_units (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE, -- e.g., 'cup', 'gram', 'tablespoon', 'piece'
    abbreviation VARCHAR(20), -- e.g., 'g', 'kg', 'tbsp', 'tsp'
    system measurement_system NOT NULL,
    base_unit_id UUID, -- For conversions (e.g., gram is base for weight)
    conversion_factor DECIMAL(10,6), -- How many base units this unit represents
    
    FOREIGN KEY (base_unit_id) REFERENCES measurement_units(id)
);

-- Recipe ingredients junction table
CREATE TABLE recipe_ingredients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipe_id UUID NOT NULL,
    ingredient_id UUID NOT NULL,
    quantity DECIMAL(10,3), -- Can be null for "to taste" items
    unit_id UUID,
    notes TEXT, -- For additional info like "chopped", "to taste", "optional"
    sort_order INTEGER NOT NULL DEFAULT 0, -- To maintain ingredient order
    
    FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON DELETE CASCADE,
    FOREIGN KEY (ingredient_id) REFERENCES ingredients(id),
    FOREIGN KEY (unit_id) REFERENCES measurement_units(id),
    
    UNIQUE(recipe_id, ingredient_id), -- Prevent duplicate ingredients per recipe
    CHECK (quantity IS NULL OR quantity > 0)
);

-- Recipe instructions/method steps
CREATE TABLE recipe_steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipe_id UUID NOT NULL,
    step_number INTEGER NOT NULL,
    instruction TEXT NOT NULL,
    duration_minutes INTEGER CHECK (duration_minutes >= 0), -- Optional timing per step
    temperature VARCHAR(50), -- e.g., "190°C", "gas mark 5"
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON DELETE CASCADE,
    UNIQUE(recipe_id, step_number)
);

-- Recipe tags for categorization and searching
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    color VARCHAR(7), -- Hex color code for UI
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE recipe_tags (
    recipe_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    
    PRIMARY KEY (recipe_id, tag_id),
    FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- Recipe ratings and reviews (optional feature)
CREATE TABLE recipe_ratings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipe_id UUID NOT NULL,
    user_id UUID, -- For user tracking
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON DELETE CASCADE,
    UNIQUE(recipe_id, user_id) -- One rating per user per recipe
);

-- Recipe collections/cookbooks
CREATE TABLE collections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_by UUID,
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE collection_recipes (
    collection_id UUID NOT NULL,
    recipe_id UUID NOT NULL,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (collection_id, recipe_id),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
    FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_recipes_title ON recipes USING GIN(to_tsvector('english', title));
CREATE INDEX idx_recipes_search_vector ON recipes USING GIN(search_vector);
CREATE INDEX idx_recipes_created_at ON recipes(created_at DESC);
CREATE INDEX idx_recipes_serves ON recipes(serves);
CREATE INDEX idx_recipes_total_time ON recipes(total_time_minutes);

CREATE INDEX idx_recipe_ingredients_recipe_id ON recipe_ingredients(recipe_id);
CREATE INDEX idx_recipe_ingredients_ingredient_id ON recipe_ingredients(ingredient_id);
CREATE INDEX idx_recipe_ingredients_sort_order ON recipe_ingredients(recipe_id, sort_order);

CREATE INDEX idx_recipe_steps_recipe_id ON recipe_steps(recipe_id);
CREATE INDEX idx_recipe_steps_step_number ON recipe_steps(recipe_id, step_number);

CREATE INDEX idx_ingredients_name ON ingredients(name);
CREATE INDEX idx_ingredients_category ON ingredients(category);

CREATE INDEX idx_measurement_units_system ON measurement_units(system);
CREATE INDEX idx_measurement_units_name ON measurement_units(name);

-- Triggers for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_recipes_updated_at 
    BEFORE UPDATE ON recipes 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_collections_updated_at 
    BEFORE UPDATE ON collections 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Sample data insertion for common measurement units
INSERT INTO measurement_units (name, abbreviation, system, conversion_factor) VALUES
-- Metric units
('gram', 'g', 'metric', 1.0),
('kilogram', 'kg', 'metric', 1000.0),
('millilitre', 'ml', 'metric', 1.0),
('litre', 'l', 'metric', 1000.0),
('piece', 'pc', 'metric', 1.0),
-- Imperial units
('cup', 'cup', 'imperial', 240.0), -- ml equivalent
('tablespoon', 'tbsp', 'imperial', 15.0), -- ml equivalent
('teaspoon', 'tsp', 'imperial', 5.0), -- ml equivalent
('ounce', 'oz', 'imperial', 28.35), -- gram equivalent
('pound', 'lb', 'imperial', 453.6), -- gram equivalent
('fluid ounce', 'fl oz', 'imperial', 29.57), -- ml equivalent
('pint', 'pt', 'imperial', 473.18), -- ml equivalent
('quart', 'qt', 'imperial', 946.35); -- ml equivalent

-- Sample ingredients
INSERT INTO ingredients (name, category) VALUES
('asparagus', 'vegetables'),
('feta cheese', 'dairy'),
('puff pastry', 'bakery'),
('olive oil', 'oils'),
('eggs', 'dairy'),
('cumin seeds', 'spices'),
('coriander seeds', 'spices'),
('nigella seeds', 'spices'),
('lemon', 'fruits'),
('greek yogurt', 'dairy'),
('black pepper', 'spices'),
('salt', 'seasoning');

-- Sample tags
INSERT INTO tags (name, description, color) VALUES
('vegetarian', 'Suitable for vegetarians', '#4CAF50'),
('quick', 'Ready in 30 minutes or less', '#FF9800'),
('Mediterranean', 'Mediterranean cuisine', '#2196F3'),
('pastry', 'Contains pastry', '#9C27B0'),
('savory tart', 'Savory tart recipes', '#795548');

-- View for easy recipe browsing with aggregated data
CREATE VIEW recipe_summary AS
SELECT 
    r.id,
    r.title,
    r.description,
    r.photo_filename,
    r.serves,
    r.prep_time_minutes,
    r.cook_time_minutes,
    r.total_time_minutes,
    r.created_at,
    r.updated_at,
    COUNT(DISTINCT ri.id) as ingredient_count,
    COUNT(DISTINCT rs.id) as step_count,
    COUNT(DISTINCT rt.tag_id) as tag_count
FROM recipes r
LEFT JOIN recipe_ingredients ri ON r.id = ri.recipe_id
LEFT JOIN recipe_steps rs ON r.id = rs.recipe_id
LEFT JOIN recipe_tags rt ON r.id = rt.recipe_id
GROUP BY r.id, r.title, r.description, r.photo_filename, r.serves, 
         r.prep_time_minutes, r.cook_time_minutes, r.total_time_minutes,
         r.created_at, r.updated_at;

-- View for full recipe details including ingredients and steps
CREATE VIEW recipe_details AS
SELECT 
    r.*,
    json_agg(
        DISTINCT jsonb_build_object(
            'ingredient_name', i.name,
            'quantity', ri.quantity,
            'unit', mu.name,
            'unit_abbr', mu.abbreviation,
            'notes', ri.notes,
            'sort_order', ri.sort_order
        ) ORDER BY ri.sort_order
    ) FILTER (WHERE i.id IS NOT NULL) as ingredients,
    json_agg(
        DISTINCT jsonb_build_object(
            'step_number', rs.step_number,
            'instruction', rs.instruction,
            'duration_minutes', rs.duration_minutes,
            'temperature', rs.temperature
        ) ORDER BY rs.step_number
    ) FILTER (WHERE rs.id IS NOT NULL) as steps,
    json_agg(
        DISTINCT jsonb_build_object(
            'tag_name', t.name,
            'tag_color', t.color
        )
    ) FILTER (WHERE t.id IS NOT NULL) as tags
FROM recipes r
LEFT JOIN recipe_ingredients ri ON r.id = ri.recipe_id
LEFT JOIN ingredients i ON ri.ingredient_id = i.id
LEFT JOIN measurement_units mu ON ri.unit_id = mu.id
LEFT JOIN recipe_steps rs ON r.id = rs.recipe_id
LEFT JOIN recipe_tags rt ON r.id = rt.recipe_id
LEFT JOIN tags t ON rt.tag_id = t.id
GROUP BY r.id;

-- Function to search recipes by text
CREATE OR REPLACE FUNCTION search_recipes(search_term TEXT)
RETURNS TABLE (
    id UUID,
    title VARCHAR(255),
    description TEXT,
    rank REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        r.id,
        r.title,
        r.description,
        ts_rank(r.search_vector, plainto_tsquery('english', search_term)) as rank
    FROM recipes r
    WHERE r.search_vector @@ plainto_tsquery('english', search_term)
       OR r.title ILIKE '%' || search_term || '%'
    ORDER BY rank DESC, r.title;
END;
$$ LANGUAGE plpgsql;