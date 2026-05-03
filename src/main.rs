mod api;
mod models;
mod repository;
mod services;

use actix_web::{web, App, HttpServer, middleware::Logger};
use handlebars::Handlebars;
use mongodb::{Client, options::ClientOptions};
use std::env;
use std::sync::Mutex;
use crate::repository::note_repository::NoteRepository;
use crate::services::note_service::NoteService;
use crate::models::game::GameState;
use crate::api::game_handler::GameData;

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

    // Shared game state
    let game_data = web::Data::new(GameData {
        state: Mutex::new(GameState::default()),
    });

    let mut hb = Handlebars::new();
    hb.register_template_file("base", "./templates/base.html").expect("Failed to register base template");
    hb.register_template_file("dashboard", "./templates/dashboard.html").expect("Failed to register dashboard template");
    hb.register_template_file("notes", "./templates/notes.html").expect("Failed to register notes template");
    hb.register_template_file("note_list", "./templates/note_list.html").expect("Failed to register note_list template");
    hb.register_template_file("note_edit_form", "./templates/note_edit_form.html").expect("Failed to register note_edit_form template");
    hb.register_template_file("jwt", "./templates/jwt.html").expect("Failed to register jwt template");
    hb.register_template_file("jwt_result", "./templates/jwt_result.html").expect("Failed to register jwt_result template");
    hb.register_template_file("tictactoe", "./templates/tictactoe.html").expect("Failed to register tictactoe template");
    hb.register_template_file("tictactoe_board", "./templates/tictactoe_board.html").expect("Failed to register tictactoe_board template");
    
    // Register helpers
    hb.register_helper("eq", Box::new(|h: &handlebars::Helper, _: &Handlebars, _: &handlebars::Context, _: &mut handlebars::RenderContext, out: &mut dyn handlebars::Output| -> handlebars::HelperResult {
        let p1 = h.param(0).and_then(|v| v.value().as_str()).unwrap_or("");
        let p2 = h.param(1).and_then(|v| v.value().as_str()).unwrap_or("");
        if p1 == p2 {
            out.write("true")?;
        }
        Ok(())
    }));

    hb.register_escape_fn(handlebars::no_escape);
    
    let hb_data = web::Data::new(hb);

    log::info!("Starting server at http://0.0.0.0:8080");

    HttpServer::new(move || {
        App::new()
            .wrap(Logger::default())
            .app_data(note_service_data.clone())
            .app_data(hb_data.clone())
            .app_data(game_data.clone())
            .service(api::note_handler::dashboard)
            .service(api::note_handler::note_index)
            .service(api::note_handler::create_note)
            .service(api::note_handler::edit_note_form)
            .service(api::note_handler::update_note)
            .service(api::note_handler::delete_note)
            .service(api::jwt_handler::jwt_index)
            .service(api::jwt_handler::decode_jwt)
            .service(api::jwt_handler::mint_jwt)
            .service(api::game_handler::game_index)
            .service(api::game_handler::make_move)
            .service(api::game_handler::reset_game)
            .service(actix_files::Files::new("/static", "./static").show_files_listing())
    })
    .bind(("0.0.0.0", 8080))?
    .run()
    .await
}
