#![allow(dead_code)]

use std::collections::HashMap;

#[derive(Debug, Clone)]
pub struct BidirectionalIndexMap {
    uuid_to_index: HashMap<String, usize>,
    index_to_uuid: Vec<String>,
}

impl BidirectionalIndexMap {
    pub fn new<'a, I>(ordered_uuids: I) -> Result<Self, String>
    where
        I: IntoIterator<Item = &'a str>,
    {
        let mut uuid_to_index = HashMap::new();
        let mut index_to_uuid = Vec::new();

        for raw_uuid in ordered_uuids {
            let uuid = raw_uuid.trim();
            if uuid.is_empty() {
                continue;
            }
            if uuid_to_index.contains_key(uuid) {
                return Err(format!("duplicate uuid in mapping input: {uuid}"));
            }
            uuid_to_index.insert(uuid.to_string(), index_to_uuid.len());
            index_to_uuid.push(uuid.to_string());
        }

        Ok(Self {
            uuid_to_index,
            index_to_uuid,
        })
    }

    pub fn size(&self) -> usize {
        self.index_to_uuid.len()
    }

    pub fn index_of(&self, uuid: &str) -> Option<usize> {
        self.uuid_to_index.get(uuid).copied()
    }

    pub fn uuid_of(&self, index: usize) -> Option<&str> {
        self.index_to_uuid.get(index).map(String::as_str)
    }

    pub fn ordered(&self) -> Vec<String> {
        self.index_to_uuid.clone()
    }
}
