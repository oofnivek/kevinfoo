mod api;
mod models;
mod repository;
mod services;

use actix_web::{web, App, HttpServer, middleware::Logger};
use handlebars::Handlebars;
use mongodb::{Client, options::ClientOptions};
use std::env;
use crate::repository::note_repository::NoteRepository;
use crate::services::note_service::NoteService;

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    dotenv::dotenv().ok();
    env_logger::init_from_env(env_logger::Env::new().default_filter_or("info"));

    let mongo_uri = env::var("MONGODB_URI").expect("MONGODB_URI must be set");
    let client_options = ClientOptions::parse(mongo_uri).await.expect("Failed to parse MongoDB URI");
    let client = Client::with_options(client_options).expect("Failed to create MongoDB client");
    let db = client.database("kevinfoo_db");

    let note_repo = NoteRepository::new(&db);
    let note_service = NoteService::new(note_repo);
    let note_service_data = web::Data::new(note_service);

    let mut hb = Handlebars::new();
    hb.register_template_file("index", "./templates/index.html").expect("Failed to register index template");
    hb.register_template_file("note_list", "./templates/note_list.html").expect("Failed to register note_list template");
    hb.register_template_file("note_edit_form", "./templates/note_edit_form.html").expect("Failed to register note_edit_form template");
    let hb_data = web::Data::new(hb);

    log::info!("Starting server at http://0.0.0.0:8080");

    HttpServer::new(move || {
        App::new()
            .wrap(Logger::default())
            .app_data(note_service_data.clone())
            .app_data(hb_data.clone())
            .service(api::note_handler::index)
            .service(api::note_handler::create_note)
            .service(api::note_handler::edit_note_form)
            .service(api::note_handler::update_note)
            .service(api::note_handler::delete_note)
            .service(actix_files::Files::new("/static", "./static").show_files_listing())
    })
    .bind(("0.0.0.0", 8080))?
    .run()
    .await
}
