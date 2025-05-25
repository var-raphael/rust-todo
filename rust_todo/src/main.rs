use serde::{Serialize, Deserialize};
use std::collections::HashMap;
use std::fs::{self, File};
use std::io::{self, Write};
use std::path::Path;

#[derive(Serialize, Deserialize, Debug)]
struct TaskManager {
    tasks: HashMap<u32, String>,
    next_id: u32,
}

impl TaskManager {
    fn new() -> Self {
        if Path::new("tasks.json").exists() {
            let data = fs::read_to_string("tasks.json").expect("Failed to read file");
            serde_json::from_str(&data).unwrap_or(Self {
                tasks: HashMap::new(),
                next_id: 1,
            })
        } else {
            Self {
                tasks: HashMap::new(),
                next_id: 1,
            }
        }
    }

    fn save(&self) {
        let json = serde_json::to_string_pretty(&self).expect("Failed to serialize tasks");
        fs::write("tasks.json", json).expect("Failed to write to file");
    }

    fn add_task(&mut self, task: String) {
        self.tasks.insert(self.next_id, task);
        println!("Task added with ID {}", self.next_id);
        self.next_id += 1;
        self.save();
    }

    fn list_tasks(&self) {
        println!("\nYour tasks:");
        for (id, task) in &self.tasks {
            println!("[{}] {}", id, task);
        }
    }

    fn remove_task(&mut self, id: u32) {
        if self.tasks.remove(&id).is_some() {
            println!("Task {} removed.", id);
        } else {
            println!("No task with ID {}", id);
        }
        self.save();
    }
}

fn main() {
    let mut manager = TaskManager::new();

    println!("Welcome to the Rust JSON To-Do App!");
    loop {
        println!("\nChoose an option:");
        println!("1. Add task");
        println!("2. List tasks");
        println!("3. Remove task");
        println!("4. Exit");

        let mut choice = String::new();
        io::stdin().read_line(&mut choice).expect("Failed to read input");
        match choice.trim() {
            "1" => {
                println!("Enter task:");
                let mut task = String::new();
                io::stdin().read_line(&mut task).expect("Failed to read task");
                manager.add_task(task.trim().to_string());
            },
            "2" => manager.list_tasks(),
            "3" => {
                println!("Enter task ID to remove:");
                let mut id_str = String::new();
                io::stdin().read_line(&mut id_str).expect("Failed to read input");
                if let Ok(id) = id_str.trim().parse::<u32>() {
                    manager.remove_task(id);
                } else {
                    println!("Invalid ID.");
                }
            },
            "4" => {
                println!("Goodbye!");
                break;
            },
            _ => println!("Invalid choice. Try again."),
        }
    }
}
