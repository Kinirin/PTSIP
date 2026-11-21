use saphyr::{Scalar, Yaml, YamlLoader};
use saphyr_parser::{Event, Parser, SpannedEventReceiver};
use serde_json::{Map, Number, Value};

// Strict bounded YAML normalization for registered Profile resources.
pub fn parse_yaml(bytes: &[u8]) -> Result<Value, String> {
    let text = std::str::from_utf8(bytes).map_err(|error| error.to_string())?;
    let mut loader = YamlLoader::<Yaml>::default();
    let mut depth = 0;
    let mut document_started = false;
    for event in Parser::new_from_str(text) {
        let (mut event, span) = event.map_err(|error| error.to_string())?;
        match &mut event {
            Event::DocumentStart(_) => {
                if document_started {
                    return Err("expected one YAML document".into());
                }
                document_started = true;
            }
            Event::Alias(_) => return Err("YAML aliases are unsupported".into()),
            Event::Scalar(_, _, anchor, _) => *anchor = 0,
            Event::MappingStart(anchor, _) | Event::SequenceStart(anchor, _) => {
                if depth == 128 {
                    return Err("YAML nesting limit exceeded".into());
                }
                depth += 1;
                *anchor = 0;
            }
            Event::MappingEnd | Event::SequenceEnd => depth -= 1,
            _ => {}
        }
        loader.on_event(event, span);
        if let Some(error) = loader.error() {
            return Err(error.to_string());
        }
    }
    let document = loader
        .into_documents()
        .pop()
        .ok_or("expected one YAML document")?;
    yaml_value(document)
}

fn yaml_value(node: Yaml<'_>) -> Result<Value, String> {
    match node {
        Yaml::Value(Scalar::Null) => Ok(Value::Null),
        Yaml::Value(Scalar::Boolean(value)) => Ok(Value::Bool(value)),
        Yaml::Value(Scalar::Integer(value)) => Ok(Value::Number(value.into())),
        Yaml::Value(Scalar::FloatingPoint(value)) => Number::from_f64(value.into_inner())
            .map(Value::Number)
            .ok_or_else(|| "non-finite YAML number".into()),
        Yaml::Value(Scalar::String(value)) => Ok(Value::String(value.into_owned())),
        Yaml::Sequence(items) => items
            .into_iter()
            .map(yaml_value)
            .collect::<Result<Vec<_>, _>>()
            .map(Value::Array),
        Yaml::Mapping(entries) => {
            let mut object = Map::new();
            for (key, value) in entries {
                let Yaml::Value(Scalar::String(key)) = key else {
                    return Err("YAML mapping keys must be strings".into());
                };
                object.insert(key.into_owned(), yaml_value(value)?);
            }
            Ok(Value::Object(object))
        }
        _ => Err("YAML node cannot be represented as JSON".into()),
    }
}
