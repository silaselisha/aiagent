use clap::{Parser, Subcommand};
use rand::prelude::*;
use rand::seq::SliceRandom;
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
        #[arg(long, default_value_t = 0.2)]
        val_split: f32,
        #[arg(long, default_value_t = 3)]
        patience: usize,
        #[arg(long)]
        checkpoint: Option<String>,
        #[arg(long, default_value_t = true)]
        calibrate: bool,
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
struct ModelFile { input: usize, hidden: usize, output: usize, mlp: MLP, threshold: f32 }

fn mse(mlp: &MLP, data: &[Sample]) -> f32 {
    if data.is_empty() { return 0.0; }
    let mut sum = 0.0f32;
    for s in data {
        let (_, y) = mlp.forward(&s.x);
        for k in 0..y.len() {
            let diff = y[k] - s.y[k];
            sum += diff * diff;
        }
    }
    sum / (data.len() as f32)
}

fn best_threshold_f1(mlp: &MLP, data: &[Sample]) -> (f32, f32) {
    if data.is_empty() { return (0.0, 0.0); }
    // collect predictions and binary labels (y>0 => positive)
    let mut preds: Vec<f32> = Vec::with_capacity(data.len());
    let mut labels: Vec<i32> = Vec::with_capacity(data.len());
    let mut min_p = f32::MAX;
    let mut max_p = f32::MIN;
    for s in data {
        let (_, y) = mlp.forward(&s.x);
        let p = y[0];
        preds.push(p);
        labels.push(if s.y[0] > 0.0 { 1 } else { 0 });
        if p < min_p { min_p = p; }
        if p > max_p { max_p = p; }
    }
    if (max_p - min_p).abs() < 1e-6 { return (min_p, 0.0); }
    let mut best_t = min_p;
    let mut best_f1 = 0.0f32;
    // scan thresholds across 50 steps
    let steps = 50;
    for i in 0..=steps {
        let t = min_p + (max_p - min_p) * (i as f32) / (steps as f32);
        let mut tp = 0; let mut fp = 0; let mut fn_ = 0;
        for idx in 0..preds.len() {
            let pred_pos = preds[idx] >= t;
            let label_pos = labels[idx] == 1;
            match (pred_pos, label_pos) {
                (true, true) => tp += 1,
                (true, false) => fp += 1,
                (false, true) => fn_ += 1,
                _ => {}
            }
        }
        let precision = if tp + fp > 0 { tp as f32 / (tp + fp) as f32 } else { 0.0 };
        let recall = if tp + fn_ > 0 { tp as f32 / (tp + fn_) as f32 } else { 0.0 };
        let f1 = if precision + recall > 0.0 { 2.0 * precision * recall / (precision + recall) } else { 0.0 };
        if f1 > best_f1 { best_f1 = f1; best_t = t; }
    }
    (best_t, best_f1)
}

fn main() -> io::Result<()> {
    let cli = Cli::parse();
    match cli.command {
        Commands::Train { out, data, hidden, epochs, lr, val_split, patience, checkpoint, calibrate } => {
            let samples = read_jsonl(&data)?;
            if samples.is_empty() { return Err(io::Error::new(io::ErrorKind::InvalidInput, "no samples")); }
            let input = samples[0].x.len();
            let output = samples[0].y.len();
            let mut rng = rand::thread_rng();
            let mut data_shuf = samples.clone();
            data_shuf.shuffle(&mut rng);
            let vsz = ((data_shuf.len() as f32) * val_split).round() as usize;
            let vsz = vsz.min(data_shuf.len().saturating_sub(1));
            let (val_set, train_set) = data_shuf.split_at(vsz);
            let mut mlp = MLP::new(input, hidden, output, &mut rng);
            let mut best_mlp = mlp.clone();
            let mut best_loss = f32::INFINITY;
            let mut bad_epochs = 0usize;
            let ckpt = checkpoint.unwrap_or_else(|| out.clone());
            for _e in 0..epochs {
                mlp.train_epoch(train_set, lr);
                let val_loss = mse(&mlp, val_set);
                if val_loss + 1e-6 < best_loss {
                    best_loss = val_loss;
                    best_mlp = mlp.clone();
                    bad_epochs = 0;
                    // save checkpoint
                    let mf_ck = ModelFile { input, hidden, output, mlp: best_mlp.clone(), threshold: 0.0 };
                    fs::write(&ckpt, serde_json::to_vec(&mf_ck).unwrap())?;
                } else {
                    bad_epochs += 1;
                    if bad_epochs >= patience { break; }
                }
            }
            // calibration threshold on validation set
            let mut threshold = 0.0f32;
            if calibrate {
                let (t, _f1) = best_threshold_f1(&best_mlp, val_set);
                threshold = t;
            }
            let mf = ModelFile { input, hidden, output, mlp: best_mlp, threshold };
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
