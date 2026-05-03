use actix_web::{get, post, put, delete, web, HttpResponse, Responder};
use crate::services::note_service::NoteService;
use crate::models::note::{CreateNoteRequest, UpdateNoteRequest};
use handlebars::Handlebars;
use serde_json::json;

#[get("/")]
pub async fn dashboard(hb: web::Data<Handlebars<'_>>) -> impl Responder {
    let body = hb.render("dashboard", &json!({})).unwrap();
    let html = hb.render("base", &json!({
        "title": "Dashboard",
        "body": body
    })).unwrap();
    HttpResponse::Ok().body(html)
}

#[get("/notes")]
pub async fn note_index(hb: web::Data<Handlebars<'_>>, service: web::Data<NoteService>) -> impl Responder {
    let notes = service.get_notes().await.unwrap_or_default();
    let data = json!({
        "notes": notes
    });
    let body = hb.render("notes", &data).unwrap();
    let html = hb.render("base", &json!({
        "title": "Note Manager",
        "body": body
    })).unwrap();
    HttpResponse::Ok().body(html)
}

#[post("/notes")]
pub async fn create_note(
    hb: web::Data<Handlebars<'_>>,
    service: web::Data<NoteService>,
    form: web::Form<CreateNoteRequest>,
) -> impl Responder {
    let _ = service.create_note(form.into_inner()).await;
    let notes = service.get_notes().await.unwrap_or_default();
    let data = json!({
        "notes": notes
    });
    let body = hb.render("note_list", &data).unwrap();
    HttpResponse::Ok().body(body)
}

#[get("/notes/{id}/edit")]
pub async fn edit_note_form(
    hb: web::Data<Handlebars<'_>>,
    service: web::Data<NoteService>,
    path: web::Path<String>,
) -> impl Responder {
    let id = path.into_inner();
    if let Ok(Some(note)) = service.get_note(&id).await {
        let body = hb.render("note_edit_form", &note).unwrap();
        HttpResponse::Ok().body(body)
    } else {
        HttpResponse::NotFound().body("Note not found")
    }
}

#[put("/notes/{id}")]
pub async fn update_note(
    hb: web::Data<Handlebars<'_>>,
    service: web::Data<NoteService>,
    path: web::Path<String>,
    form: web::Form<UpdateNoteRequest>,
) -> impl Responder {
    let id = path.into_inner();
    let _ = service.update_note(&id, form.into_inner()).await;
    let notes = service.get_notes().await.unwrap_or_default();
    let data = json!({
        "notes": notes
    });
    let body = hb.render("note_list", &data).unwrap();
    HttpResponse::Ok().body(body)
}

#[delete("/notes/{id}")]
pub async fn delete_note(
    hb: web::Data<Handlebars<'_>>,
    service: web::Data<NoteService>,
    path: web::Path<String>,
) -> impl Responder {
    let id = path.into_inner();
    let _ = service.delete_note(&id).await;
    let notes = service.get_notes().await.unwrap_or_default();
    let data = json!({
        "notes": notes
    });
    let body = hb.render("note_list", &data).unwrap();
    HttpResponse::Ok().body(body)
}
