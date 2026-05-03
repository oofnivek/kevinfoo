use actix_web::{get, post, delete, web, HttpResponse, Responder};
use crate::services::task_service::TaskService;
use crate::models::task::CreateTaskRequest;
use handlebars::Handlebars;
use serde_json::json;

#[get("/")]
pub async fn index(hb: web::Data<Handlebars<'_>>, service: web::Data<TaskService>) -> impl Responder {
    let tasks = service.get_tasks().await.unwrap_or_default();
    let data = json!({
        "tasks": tasks
    });
    let body = hb.render("index", &data).unwrap();
    HttpResponse::Ok().body(body)
}

#[post("/tasks")]
pub async fn create_task(
    hb: web::Data<Handlebars<'_>>,
    service: web::Data<TaskService>,
    form: web::Form<CreateTaskRequest>,
) -> impl Responder {
    let _ = service.create_task(form.into_inner()).await;
    let tasks = service.get_tasks().await.unwrap_or_default();
    let data = json!({
        "tasks": tasks
    });
    let body = hb.render("task_list", &data).unwrap();
    HttpResponse::Ok().body(body)
}

#[post("/tasks/{id}/toggle")]
pub async fn toggle_task(
    hb: web::Data<Handlebars<'_>>,
    service: web::Data<TaskService>,
    path: web::Path<String>,
) -> impl Responder {
    let id = path.into_inner();
    let _ = service.toggle_task(&id).await;
    let tasks = service.get_tasks().await.unwrap_or_default();
    let data = json!({
        "tasks": tasks
    });
    let body = hb.render("task_list", &data).unwrap();
    HttpResponse::Ok().body(body)
}

#[delete("/tasks/{id}")]
pub async fn delete_task(
    hb: web::Data<Handlebars<'_>>,
    service: web::Data<TaskService>,
    path: web::Path<String>,
) -> impl Responder {
    let id = path.into_inner();
    let _ = service.delete_task(&id).await;
    let tasks = service.get_tasks().await.unwrap_or_default();
    let data = json!({
        "tasks": tasks
    });
    let body = hb.render("task_list", &data).unwrap();
    HttpResponse::Ok().body(body)
}
