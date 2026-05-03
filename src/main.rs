mod api;
mod models;
mod repository;
mod services;

use actix_web::{web, App, HttpServer, middleware::Logger};
use handlebars::Handlebars;
use mongodb::{Client, options::ClientOptions};
use std::env;
use crate::repository::task_repository::TaskRepository;
use crate::services::task_service::TaskService;

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    dotenv::dotenv().ok();
    env_logger::init_from_env(env_logger::Env::new().default_filter_or("info"));

    let mongo_uri = env::var("MONGODB_URI").expect("MONGODB_URI must be set");
    let client_options = ClientOptions::parse(mongo_uri).await.expect("Failed to parse MongoDB URI");
    let client = Client::with_options(client_options).expect("Failed to create MongoDB client");
    let db = client.database("kevinfoo_db");

    let task_repo = TaskRepository::new(&db);
    let task_service = TaskService::new(task_repo);
    let task_service_data = web::Data::new(task_service);

    let mut hb = Handlebars::new();
    hb.register_template_file("index", "./templates/index.html").expect("Failed to register index template");
    hb.register_template_file("task_list", "./templates/task_list.html").expect("Failed to register task_list template");
    let hb_data = web::Data::new(hb);

    log::info!("Starting server at http://0.0.0.0:8080");

    HttpServer::new(move || {
        App::new()
            .wrap(Logger::default())
            .app_data(task_service_data.clone())
            .app_data(hb_data.clone())
            .service(api::task_handler::index)
            .service(api::task_handler::create_task)
            .service(api::task_handler::toggle_task)
            .service(api::task_handler::delete_task)
            .service(actix_files::Files::new("/static", "./static").show_files_listing())
    })
    .bind(("0.0.0.0", 8080))?
    .run()
    .await
}
