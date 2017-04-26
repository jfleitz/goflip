/* 
 JAF 2017-04-16: Solenoid Driver controller. Now using USB on a Arduino Nano


This will drive the SDU board using an Arduino Nano.
To communicate, the NAK is set to "c". 
Message is in the following format:

{[SolID][Duration/Cmd]}

Duration/Cmd is as followed:
0 = off (allows to force the off)
1 = pulse (default time, whatever a default pulse is (50ms?))
2-254 (02-0xfe) = 10ms*the value passed in (so up to 2.54 sec)

SolID 16 = on board LED


If SolID is > 16, then it is considered a "Direct Control" as followed:

SolID   |  Control
-----------------------------
17      |  Turns on Relay1
18      |  Turns on Relay 2
19      |  Turns on J2/10
20      |  Turns on J2/9


Send a NAK when ACK is receieved ("|") 


 
 
 P2 
 1 = Relay 1 (active high)
 2 = Relay 2 (active high)
 3 = J2/10  (active high)
 4 = J2/9 (active high)
 5 = Addr 1
 6 = Addr 2
 7 = Key
 8 = Addr 3
 9 = Addr 4

Nano:
Low PortD (Active High)
 4=sol Address 0 
 5=sol Address 1
 6=sol Address 2
 7=sol Address 3
 
 Low PortB (Active High as well):
 0=str Relay 1
 1=str Relay 2
 2=str J2/10
 3=str J2/9
 */

#include <avr/io.h>
//#define DEBUG true
#define F_CPU 16000000UL  //16mhz clock
#include <util/delay.h> 
#define SDU_NAK 0x63 //  c

// Pin 13 has an LED connected on most Arduino boards.
// PB5 is the LED
// give it a name:
int led = 13; 

volatile boolean decrement;

//holds the status of the lamp.
//00=off, 01=on, 10=slow blink, 11=fast blink
volatile unsigned char _solControl[17]; 
volatile byte _lastSol; //last Lamp ID command sent. Makes debugging easier


enum recvstate {
  SolID = 1,
  ControlID,
  EOL
};


boolean ack = false;

// the setup routine runs once when you press reset:
void setup() {                
  // initialize the digital pin as an output.
  pinMode(led, OUTPUT);     
  Serial.begin(57600); // start serial for output. Should only have this if debugging
  _lastSol = 0;
  decrement = false;
  SetSolAll(0);

  DDRD = DDRD | B11110000; //Port D4-D7 output for Lamp0-Lamp3
  DDRB = DDRB | B00001111; //Port B0-B3 output for Lamp Strobe0-Strobe3

  // initialize timer1 
  noInterrupts();           // disable all interrupts
  TCCR1A = 0;
  TCCR1B = 0;
  TCNT1  = 0;

  OCR1A =  625;           // compare match register 16MHz/256/100Hz 
  TCCR1B |= (1 << WGM12);   // CTC mode
  TCCR1B |= (1 << CS12);    // 256 prescaler 
  TIMSK1 |= (1 << OCIE1A);  // enable timer compare interrupt
  interrupts();             // enable all interrupts

  #ifdef DEBUG
  Serial.println("==SDU initialized==");
  #endif
  
  ack=true;
  
 // Serial.println("Initialized!");
}

void SetSolAll(int value){
  int i;

  for(i=0;i<=16;i++){
    _solControl[i]=value;
  }
}

void SetSolenoid(int sol, unsigned char data){
  //0=off, 1=on, 2=slow blink, 3=fast blink, 4=brightness
  _solControl[sol]=data;
}

// the loop routine runs over and over again forever:
void loop() {
  int i;
  boolean dodecrement=false;
  
  if(ack){
    Serial.write(SDU_NAK);
    ack=false;
  }
  
  /*
   Low PortD:
   4=lamp0
   5=lamp1
   6=lamp2
   7=lamp3
   
   Low PortB:
   0=str0
   1=str1
   2=str2
   3=str3
   */

    //just loop through each lamp value...
    if(decrement){
      decrement=false;
      dodecrement=true;
      
    } else {
      dodecrement = false ;
    }
  
  turnOffSols();
  
    for(i=0;i<=16;i++){
      if(_solControl[i] >0) {
        
        if(i<16){
         turnOnSol(i);
        }else{
          turnOnLED();
        }
                
         //step it down if we are ready (unless set to always on which is 'z')
         if(dodecrement && _solControl[i]<74){
           
           _solControl[i]--;
           Serial.print("decrement is now");
           Serial.print(_solControl[i]);
         }
         
      }else{
       if(i==16){
         turnOffLED();
       } 
      }
    }
    
    
    
}



ISR(TIMER1_COMPA_vect)          // timer compare interrupt service routine
{  
  decrement=true;
  /*for(i=0;i<=16;i++){
    if(_solControl[i] >0 && _solControl[i]<74){
        _solControl[i]--;
    }
  }*/
}


void serialEvent() {
  static recvstate _receiveState;
 static  byte solCommand;
  static byte solID;
  static bool received = false;
  static bool completed = false;
  
  while (Serial.available()) {
    received = true;
    //_receiveState
     
    char inChar = Serial.read(); 

    switch(inChar){
      case '{':
        _receiveState = SolID; //reset back to waiting for the LampID              
        completed = false;      
      break;
      case '}':
        completed = true;
        _receiveState = EOL;
        break;
      case '|':
        ack = true;
        break;
      default:
        switch(_receiveState){
          case SolID:
            _receiveState = ControlID;
            solID =(byte)inChar;
            
          #ifdef DEBUG
            Serial.print("solID Set to:");
            Serial.println(solID,DEC);
          #endif
            break;
          case ControlID:
          
            _receiveState = EOL;
            solCommand = (byte)inChar;
            
          #ifdef DEBUG
            Serial.print("Command set to:");
            Serial.println(solCommand,DEC);
          #endif
          
            break;
         // default:
           // _receiveState = BEGIN;
        }  
    }
    
    
  }
  
  
    if(received){
      if(completed){
        boolean skip = false;
        
        completed = false;
        
     //   _lastLamp = lampID;
        #ifdef DEBUG
         Serial.println("==EOL/Completed received===");

         // Serial.print("_lastLamp=");
         // Serial.println(_lastLamp);
          
          Serial.print("solID=");
          Serial.println(solID);
          
          Serial.print("solCommand=");
          Serial.println(solCommand);
        #endif
        
        //verify that we have good solIDs. If so, then we set LastLamp and proceed.
        if(solID >= 48 && solID<=69){
          _lastSol = solID - 48;
        }else{
          skip  =true;
        }
        
        if(solCommand >= 48 and solCommand <= 122 ){
            solCommand = solCommand - 48;
        }else{
          skip = true;
        }
        
        if(!skip){
          #ifdef DEBUG          
          Serial.println("==Not Skipping===");
          Serial.print("_lastSol=");
          Serial.println(_lastSol);
          
          Serial.print("solCommand=");
          Serial.println(solCommand); 
          #endif
          
          ack=true;
          
          
          if(_lastSol <= 16){
              _solControl[_lastSol] = solCommand;
          }else{
             //This is direct port.
             directSols(_lastSol,solCommand);
          }
        }
      }
    }
    

}


/*
 J1 
Low PortD (Active High)
 4=sol Address 0 
 5=sol Address 1
 6=sol Address 2
 7=sol Address 3
*/

void turnOnSol(int solID){
  unsigned char newValue;

  newValue = PORTD | 0xf0;  //reset high nibble to all 1111####
  newValue = newValue & (solID<<4 | 0x0f); //example for 0xE0, this would be 0x0f AND-ing with. so new value would be 0000####
  PORTD = newValue;
}

void turnOnLED(){
  //          Serial.println ("Turning On LED");
  //          PORTB ^=(1<<PB5);
  PORTB = PORTB | 0x20; //turn on PB5 = LED 
  //          _delay_ms(500); 

}
void turnOffLED(){
  PORTB = PORTB & 0xdf; //turn off PB5 = LED   
}

void turnOffSols(){
  PORTD |= 0xf0; //turn "off" the high bits.   
}

void turnOffAll(){
  PORTD |= 0xf0; //turn "off" the high bits.   
  PORTB &= 0xf0; //turn off the low bits
}

/*
 Low PortB (Active High as well):
 0=str Relay 1
 1=str Relay 2
 2=str J2/10
 3=str J2/9

SolID   |  Control
-----------------------------
17      |  Turns on Relay1
18      |  Turns on Relay 2
19      |  Turns on J2/10
20      |  Turns on J2/9
*/
void directSols(byte solID, byte cmd){
  #ifdef DEBUG
    Serial.print("directSols called for:");
    Serial.println(solID,DEC);
    Serial.println(cmd,DEC);
    Serial.println("----");
    
  #endif
  if(cmd>0){
    switch(solID){
      case 17:  //Relay1
        PORTB|=0x01;
        break;
      case 18:  //Relay2
         PORTB|=0x02;
         break; 
      case 19:  //J2/10
        PORTB|=0x04;
        break;
      case 20:  //J2/9
        PORTB|=0x08;
        break;
    }
  }else{
    switch(solID){
      case 17:  //Relay1
        PORTB&=0xfe;
        break;
      case 18:  //Relay2
        PORTB&=0xfd;
        break; 
      case 19:  //J2/10
        PORTB&=0xfb;
        break;
      case 20:  //J2/9
        PORTB&=0xf7;
        break;
    }    
  }
}
