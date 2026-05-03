use actix_web::{get, post, web, HttpResponse, Responder};
use handlebars::Handlebars;
use serde::Deserialize;
use serde_json::json;
use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine as _};

#[derive(Deserialize)]
pub struct DecodeRequest {
    pub token: String,
}

#[get("/jwt")]
pub async fn jwt_index(hb: web::Data<Handlebars<'_>>) -> impl Responder {
    let body = hb.render("jwt", &json!({})).unwrap();
    let html = hb.render("base", &json!({
        "title": "JWT Decoder",
        "body": body
    })).unwrap();
    HttpResponse::Ok().body(html)
}

#[post("/jwt/decode")]
pub async fn decode_jwt(
    hb: web::Data<Handlebars<'_>>,
    form: web::Form<DecodeRequest>,
) -> impl Responder {
    let parts: Vec<&str> = form.token.split('.').collect();
    
    if parts.len() < 2 {
        let body = hb.render("jwt_result", &json!({
            "header": "",
            "payload": "",
            "error": "Invalid JWT format. Must have at least 2 segments."
        })).unwrap();
        return HttpResponse::Ok().body(body);
    }

    let decode = |s: &str| {
        URL_SAFE_NO_PAD.decode(s)
            .ok()
            .and_then(|b| String::from_utf8(b).ok())
            .and_then(|s| serde_json::from_str::<serde_json::Value>(&s).ok())
            .map(|v| serde_json::to_string_pretty(&v).unwrap_or_default())
            .unwrap_or_else(|| "Failed to decode segment".to_string())
    };

    let header = decode(parts[0]);
    let payload = decode(parts[1]);

    let body = hb.render("jwt_result", &json!({
        "header": header,
        "payload": payload
    })).unwrap();
    
    HttpResponse::Ok().body(body)
}
