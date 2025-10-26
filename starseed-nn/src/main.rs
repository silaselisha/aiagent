use clap::{Parser, Subcommand};
use rand::prelude::*;
use serde::{Deserialize, Serialize};
use std::fs;
use std::io::{self, Read};

#[derive(Debug, Parser)]
#[command(name = "starseed-nn", about = "NN for 15-min windows: train/infer")] 
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Debug, Subcommand)]
enum Commands {
    /// Train MLP on 15-min window features
    Train {
        #[arg(long, default_value = "model.json")]
        out: String,
        #[arg(long)]
        data: Option<String>, // path to training data JSONL, or stdin if not set
        #[arg(long, default_value_t = 64)]
        hidden: usize,
        #[arg(long, default_value_t = 10)]
        epochs: usize,
        #[arg(long, default_value_t = 0.01)]
        lr: f32,
    },
    /// Infer predictions for 15-min window features
    Infer {
        #[arg(long, default_value = "model.json")]
        model: String,
        #[arg(long)]
        data: Option<String>, // path to JSONL input, or stdin if not set
    },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Sample { x: Vec<f32>, y: Vec<f32> }

#[derive(Debug, Clone, Serialize, Deserialize)]
struct MLP { w1: Vec<Vec<f32>>, b1: Vec<f32>, w2: Vec<Vec<f32>>, b2: Vec<f32> }

impl MLP {
    fn new(input: usize, hidden: usize, output: usize, rng: &mut impl Rng) -> Self {
        let mut w1 = vec![vec![0.0; hidden]; input];
        for i in 0..input { for j in 0..hidden { w1[i][j] = (rng.gen::<f32>() - 0.5) * 0.1; } }
        let b1 = vec![0.0; hidden];
        let mut w2 = vec![vec![0.0; output]; hidden];
        for i in 0..hidden { for j in 0..output { w2[i][j] = (rng.gen::<f32>() - 0.5) * 0.1; } }
        let b2 = vec![0.0; output];
        Self { w1, b1, w2, b2 }
    }

    fn forward(&self, x: &[f32]) -> (Vec<f32>, Vec<f32>) {
        let h: Vec<f32> = (0..self.b1.len())
            .map(|j| {
                let mut s = self.b1[j];
                for i in 0..x.len() { s += x[i] * self.w1[i][j]; }
                s.max(0.0) // ReLU
            })
            .collect();
        let mut y = vec![0.0; self.b2.len()];
        for k in 0..self.b2.len() {
            let mut s = self.b2[k];
            for j in 0..h.len() { s += h[j] * self.w2[j][k]; }
            y[k] = s;
        }
        (h, y)
    }

    fn train_epoch(&mut self, data: &[Sample], lr: f32) {
        for sample in data {
            let x = &sample.x;
            let t = &sample.y;
            let (h, y) = self.forward(x);
            // MSE loss gradient
            let mut dy = vec![0.0; y.len()];
            for k in 0..y.len() { dy[k] = 2.0 * (y[k] - t[k]); }
            // backprop to w2, b2
            let mut db2 = vec![0.0; self.b2.len()];
            let mut dw2 = vec![vec![0.0; self.b2.len()]; self.b1.len()];
            for k in 0..self.b2.len() {
                db2[k] += dy[k];
                for j in 0..self.b1.len() { dw2[j][k] += dy[k] * h[j]; }
            }
            // backprop to h
            let mut dh = vec![0.0; self.b1.len()];
            for j in 0..self.b1.len() {
                for k in 0..self.b2.len() { dh[j] += dy[k] * self.w2[j][k]; }
            }
            // apply ReLU grad
            for j in 0..dh.len() { if h[j] <= 0.0 { dh[j] = 0.0; } }
            // backprop to w1, b1
            let mut db1 = vec![0.0; self.b1.len()];
            let mut dw1 = vec![vec![0.0; self.b1.len()]; x.len()];
            for j in 0..self.b1.len() {
                db1[j] += dh[j];
                for i in 0..x.len() { dw1[i][j] += dh[j] * x[i]; }
            }
            // SGD update
            for j in 0..self.b1.len() {
                self.b1[j] -= lr * db1[j];
                for i in 0..x.len() { self.w1[i][j] -= lr * dw1[i][j]; }
            }
            for k in 0..self.b2.len() {
                self.b2[k] -= lr * db2[k];
                for j in 0..self.b1.len() { self.w2[j][k] -= lr * dw2[j][k]; }
            }
        }
    }
}

fn read_jsonl(path: &Option<String>) -> io::Result<Vec<Sample>> {
    let mut out = Vec::new();
    let data = match path {
        Some(p) => fs::read_to_string(p)?,
        None => {
            let mut buf = String::new();
            io::stdin().read_to_string(&mut buf)?;
            buf
        }
    };
    for line in data.lines() {
        if line.trim().is_empty() { continue; }
        let s: Sample = serde_json::from_str(line).map_err(|e| io::Error::new(io::ErrorKind::InvalidData, e))?;
        out.push(s);
    }
    Ok(out)
}

#[derive(Serialize, Deserialize)]
struct ModelFile { input: usize, hidden: usize, output: usize, mlp: MLP }

fn main() -> io::Result<()> {
    let cli = Cli::parse();
    match cli.command {
        Commands::Train { out, data, hidden, epochs, lr } => {
            let samples = read_jsonl(&data)?;
            if samples.is_empty() { return Err(io::Error::new(io::ErrorKind::InvalidInput, "no samples")); }
            let input = samples[0].x.len();
            let output = samples[0].y.len();
            let mut rng = rand::thread_rng();
            let mut mlp = MLP::new(input, hidden, output, &mut rng);
            for _ in 0..epochs { mlp.train_epoch(&samples, lr); }
            let mf = ModelFile { input, hidden, output, mlp };
            fs::write(out, serde_json::to_vec(&mf).unwrap())?;
        }
        Commands::Infer { model, data } => {
            let bytes = fs::read(model)?;
            let mf: ModelFile = serde_json::from_slice(&bytes).unwrap();
            let samples = read_jsonl(&data)?;
            for s in samples {
                let (_, y) = mf.mlp.forward(&s.x);
                println!("{}", serde_json::to_string(&y).unwrap());
            }
        }
    }
    Ok(())
}
